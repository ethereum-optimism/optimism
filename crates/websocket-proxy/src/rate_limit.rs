use std::collections::HashMap;
use std::net::IpAddr;
use std::sync::{Arc, Mutex};
use tracing::{debug, error, warn};

use thiserror::Error;
use tokio::sync::{OwnedSemaphorePermit, Semaphore};

use redis::{Client, Commands, RedisError};
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::{Duration, SystemTime};
use uuid::Uuid;

#[derive(Error, Debug)]
pub enum RateLimitError {
    #[error("Rate Limit Reached: {reason}")]
    Limit { reason: String },
}

#[clippy::has_significant_drop]
pub struct Ticket {
    addr: IpAddr,
    app: Option<String>,
    _permit: OwnedSemaphorePermit,
    rate_limiter: Arc<dyn RateLimit>,
}

impl Drop for Ticket {
    fn drop(&mut self) {
        self.rate_limiter.release(self.addr, self.app.clone())
    }
}

pub trait RateLimit: Send + Sync {
    fn try_acquire(
        self: Arc<Self>,
        addr: IpAddr,
        app: Option<String>,
    ) -> Result<Ticket, RateLimitError>;

    fn release(&self, addr: IpAddr, app: Option<String>);
}

struct Inner {
    active_connections_per_ip: HashMap<IpAddr, usize>,
    active_connections_per_app: HashMap<String, usize>,
    semaphore: Arc<Semaphore>,
}

pub struct InMemoryRateLimit {
    per_ip_limit: usize,
    per_app_limit: HashMap<String, usize>,
    inner: Mutex<Inner>,
}

impl InMemoryRateLimit {
    pub fn new(
        instance_limit: usize,
        per_ip_limit: usize,
        per_app_limit: HashMap<String, usize>,
    ) -> Self {
        Self {
            per_ip_limit,
            per_app_limit,
            inner: Mutex::new(Inner {
                active_connections_per_ip: HashMap::new(),
                active_connections_per_app: HashMap::new(),
                semaphore: Arc::new(Semaphore::new(instance_limit)),
            }),
        }
    }
}

impl RateLimit for InMemoryRateLimit {
    fn try_acquire(
        self: Arc<Self>,
        addr: IpAddr,
        app: Option<String>,
    ) -> Result<Ticket, RateLimitError> {
        let mut inner = self.inner.lock().unwrap();

        let permit =
            inner
                .semaphore
                .clone()
                .try_acquire_owned()
                .map_err(|_| RateLimitError::Limit {
                    reason: "Global limit".to_owned(),
                })?;

        if self.per_ip_limit > 0 {
            let current_count = *inner.active_connections_per_ip.get(&addr).unwrap_or(&0);

            if current_count + 1 > self.per_ip_limit {
                debug!(
                    message = "Rate limit exceeded, trying to acquire",
                    client = addr.to_string()
                );
                return Err(RateLimitError::Limit {
                    reason: String::from("IP limit exceeded"),
                });
            }

            let new_count = current_count + 1;
            inner.active_connections_per_ip.insert(addr, new_count);
        }

        if let Some(app) = app.clone() {
            let current_count = *inner.active_connections_per_app.get(&app).unwrap_or(&0);

            if current_count + 1 > *self.per_app_limit.get(&app).unwrap_or(&0) {
                debug!(
                    message = "Rate limit exceeded, trying to acquire",
                    client = addr.to_string()
                );
                return Err(RateLimitError::Limit {
                    reason: String::from("App limit exceeded"),
                });
            }

            let new_count = current_count + 1;
            inner.active_connections_per_app.insert(app, new_count);
        }

        Ok(Ticket {
            addr,
            app,
            _permit: permit,
            rate_limiter: self.clone(),
        })
    }

    fn release(&self, addr: IpAddr, app: Option<String>) {
        let mut inner = self.inner.lock().unwrap();

        if self.per_ip_limit > 0 {
            let current_count = *inner.active_connections_per_ip.get(&addr).unwrap_or(&0);

            match current_count {
                0 => {
                    warn!(
                        message = "ip counting is not accurate -- unexpected underflow",
                        client = addr.to_string()
                    );
                    inner.active_connections_per_ip.remove(&addr);
                }
                1 => {
                    inner.active_connections_per_ip.remove(&addr);
                }
                _ => {
                    inner
                        .active_connections_per_ip
                        .insert(addr, current_count - 1);
                }
            }
        }

        if let Some(app) = app {
            let current_count = *inner.active_connections_per_app.get(&app).unwrap_or(&0);

            match current_count {
                0 => {
                    warn!(
                        message = "app counting is not accurate -- unexpected underflow",
                        client = app
                    );
                    inner.active_connections_per_app.remove(&app);
                }
                1 => {
                    inner.active_connections_per_app.remove(&app);
                }
                _ => {
                    inner
                        .active_connections_per_app
                        .insert(app, current_count - 1);
                }
            }
        }
    }
}

pub struct RedisRateLimit {
    redis_client: Client,
    instance_limit: usize,
    per_ip_limit: usize,
    per_app_limit: HashMap<String, usize>,
    semaphore: Arc<Semaphore>,
    key_prefix: String,
    instance_id: String,
    heartbeat_interval: Duration,
    heartbeat_ttl: Duration,
    background_tasks_started: AtomicBool,
}

impl RedisRateLimit {
    pub fn new(
        redis_url: &str,
        instance_limit: usize,
        per_ip_limit: usize,
        per_app_limit: HashMap<String, usize>,
        key_prefix: &str,
    ) -> Result<Self, RedisError> {
        let client = Client::open(redis_url)?;
        let instance_id = Uuid::new_v4().to_string();

        let heartbeat_interval = Duration::from_secs(10);
        let heartbeat_ttl = Duration::from_secs(30);

        let rate_limiter = Self {
            redis_client: client,
            instance_limit,
            per_ip_limit,
            per_app_limit,
            semaphore: Arc::new(Semaphore::new(instance_limit)),
            key_prefix: key_prefix.to_string(),
            instance_id,
            heartbeat_interval,
            heartbeat_ttl,
            background_tasks_started: AtomicBool::new(false),
        };

        if let Err(e) = rate_limiter.register_instance() {
            error!(
                message = "Failed to register instance in Redis",
                error = e.to_string()
            );
        }

        Ok(rate_limiter)
    }

    pub fn start_background_tasks(self: Arc<Self>) {
        if self.background_tasks_started.swap(true, Ordering::SeqCst) {
            return;
        }

        debug!(
            message = "Starting background heartbeat and cleanup tasks",
            instance_id = self.instance_id
        );

        let self_clone = self.clone();
        tokio::spawn(async move {
            loop {
                if let Err(e) = self_clone.update_heartbeat() {
                    error!(
                        message = "Failed to update heartbeat in background task",
                        error = e.to_string()
                    );
                }

                if let Err(e) = self_clone.cleanup_stale_instances() {
                    error!(
                        message = "Failed to cleanup stale instances in background task",
                        error = e.to_string()
                    );
                }

                tokio::time::sleep(self_clone.heartbeat_interval / 2).await;
            }
        });
    }

    fn register_instance(&self) -> Result<(), RedisError> {
        self.update_heartbeat()?;
        debug!(
            message = "Registered instance in Redis",
            instance_id = self.instance_id
        );

        Ok(())
    }

    fn update_heartbeat(&self) -> Result<(), RedisError> {
        let now = SystemTime::now();
        let mut conn = self.redis_client.get_connection()?;

        let ttl = self.heartbeat_ttl.as_secs();
        conn.set_ex::<_, _, ()>(
            self.instance_heartbeat_key(),
            now.duration_since(SystemTime::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
            ttl,
        )?;

        debug!(
            message = "Updated instance heartbeat",
            instance_id = self.instance_id
        );

        Ok(())
    }

    fn cleanup_stale_instances(&self) -> Result<(), RedisError> {
        let mut conn = self.redis_client.get_connection()?;

        let instance_heartbeat_pattern = format!("{}:instance:*:heartbeat", self.key_prefix);
        let instance_heartbeats: Vec<String> = conn.keys(instance_heartbeat_pattern)?;

        let active_instance_ids: Vec<String> = instance_heartbeats
            .iter()
            .filter_map(|key| key.split(':').nth(2).map(String::from))
            .collect();

        debug!(
            message = "Active instances with heartbeats",
            instance_count = active_instance_ids.len(),
            current_instance = self.instance_id
        );

        let ip_instance_pattern = format!("{}:ip:*:instance:*:connections", self.key_prefix);
        let ip_instance_keys: Vec<String> = conn.keys(ip_instance_pattern)?;

        let mut instance_ids_with_ip_connections = std::collections::HashSet::new();
        for key in &ip_instance_keys {
            if let Some(instance_id) = key.split(':').nth(4) {
                instance_ids_with_ip_connections.insert(instance_id.to_string());
            }
        }

        let app_instance_pattern = format!("{}:app:*:instance:*:connections", self.key_prefix);
        let app_instance_keys: Vec<String> = conn.keys(app_instance_pattern)?;

        let mut instance_ids_with_app_connections = std::collections::HashSet::new();
        for key in &app_instance_keys {
            if let Some(instance_id) = key.split(':').nth(4) {
                instance_ids_with_app_connections.insert(instance_id.to_string());
            }
        }

        debug!(
            message = "Checking for stale instances",
            instances_with_ip_connections = instance_ids_with_ip_connections.len(),
            instances_with_app_connections = instance_ids_with_app_connections.len(),
            current_instance = self.instance_id
        );

        for instance_id in instance_ids_with_ip_connections {
            if instance_id == self.instance_id {
                debug!(
                    message = "Skipping current instance",
                    instance_id = instance_id
                );
                continue;
            }

            if !active_instance_ids.contains(&instance_id) {
                debug!(
                    message = "Found stale instance",
                    instance_id = instance_id,
                    reason = "Heartbeat key not found"
                );
                self.cleanup_instance(&mut conn, &instance_id)?;
            }
        }

        for instance_id in instance_ids_with_app_connections {
            if instance_id == self.instance_id {
                debug!(
                    message = "Skipping current instance",
                    instance_id = instance_id
                );
                continue;
            }

            if !active_instance_ids.contains(&instance_id) {
                debug!(
                    message = "Found stale instance",
                    instance_id = instance_id,
                    reason = "Heartbeat key not found"
                );
                self.cleanup_instance(&mut conn, &instance_id)?;
            }
        }

        debug!(message = "Completed stale instance cleanup");

        Ok(())
    }

    fn cleanup_instance(
        &self,
        conn: &mut redis::Connection,
        instance_id: &str,
    ) -> Result<(), RedisError> {
        let ip_instance_pattern = format!(
            "{}:ip:*:instance:{}:connections",
            self.key_prefix, instance_id
        );
        let ip_instance_keys: Vec<String> = conn.keys(ip_instance_pattern)?;

        let app_instance_pattern = format!(
            "{}:app:*:instance:{}:connections",
            self.key_prefix, instance_id
        );
        let app_instance_keys: Vec<String> = conn.keys(app_instance_pattern)?;

        debug!(
            message = "Cleaning up instance",
            instance_id = instance_id,
            ip_key_count = ip_instance_keys.len(),
            app_key_count = app_instance_keys.len()
        );

        for key in ip_instance_keys {
            conn.del::<_, ()>(&key)?;
            debug!(message = "Deleted IP instance key", key = key);
        }

        for key in app_instance_keys {
            conn.del::<_, ()>(&key)?;
            debug!(message = "Deleted app instance key", key = key);
        }

        Ok(())
    }

    fn ip_instance_key(&self, addr: &IpAddr) -> String {
        format!(
            "{}:ip:{}:instance:{}:connections",
            self.key_prefix, addr, self.instance_id
        )
    }

    fn app_instance_key(&self, app: &str) -> String {
        format!(
            "{}:app:{}:instance:{}:connections",
            self.key_prefix, app, self.instance_id
        )
    }

    fn instance_heartbeat_key(&self) -> String {
        format!(
            "{}:instance:{}:heartbeat",
            self.key_prefix, self.instance_id
        )
    }
}

impl RateLimit for RedisRateLimit {
    fn try_acquire(
        self: Arc<Self>,
        addr: IpAddr,
        app: Option<String>,
    ) -> Result<Ticket, RateLimitError> {
        self.clone().start_background_tasks();

        let permit = match self.semaphore.clone().try_acquire_owned() {
            Ok(permit) => permit,
            Err(_) => {
                return Err(RateLimitError::Limit {
                    reason: "Maximum connection limit reached for this server instance".to_string(),
                });
            }
        };

        let mut conn = match self.redis_client.get_connection() {
            Ok(conn) => conn,
            Err(e) => {
                error!(
                    message = "Failed to connect to Redis",
                    error = e.to_string()
                );
                return Err(RateLimitError::Limit {
                    reason: "Redis connection failed".to_string(),
                });
            }
        };

        let mut ip_instance_connections: usize = 0;
        let mut app_instance_connections: usize = 0;
        let mut total_ip_connections: usize = 0;
        let mut total_app_connections: usize = 0;

        if self.per_ip_limit > 0 {
            let ip_keys_pattern = format!("{}:ip:{}:instance:*:connections", self.key_prefix, addr);
            let ip_keys: Vec<String> = match conn.keys(ip_keys_pattern) {
                Ok(keys) => keys,
                Err(e) => {
                    error!(
                        message = "Failed to get IP instance keys from Redis",
                        error = e.to_string()
                    );
                    return Err(RateLimitError::Limit {
                        reason: "Redis operation failed".to_string(),
                    });
                }
            };

            for key in &ip_keys {
                let count: usize = conn.get(key).unwrap_or(0);
                total_ip_connections += count;
            }

            if total_ip_connections >= self.per_ip_limit {
                return Err(RateLimitError::Limit {
                    reason: format!("Per-IP connection limit reached for {addr}"),
                });
            }

            ip_instance_connections = match conn.incr(self.ip_instance_key(&addr), 1) {
                Ok(count) => count,
                Err(e) => {
                    error!(
                        message = "Failed to increment per-instance IP counter in Redis",
                        error = e.to_string()
                    );
                    return Err(RateLimitError::Limit {
                        reason: "Redis operation failed".to_string(),
                    });
                }
            };
        }

        if let Some(app) = app.clone() {
            let app_keys_pattern =
                format!("{}:app:{}:instance:*:connections", self.key_prefix, app);
            let app_keys: Vec<String> = match conn.keys(app_keys_pattern) {
                Ok(keys) => keys,
                Err(e) => {
                    error!(
                        message = "Failed to get app instance keys from Redis",
                        error = e.to_string()
                    );
                    return Err(RateLimitError::Limit {
                        reason: "Redis operation failed".to_string(),
                    });
                }
            };

            for key in &app_keys {
                let count: usize = conn.get(key).unwrap_or(0);
                total_app_connections += count;
            }

            if total_app_connections >= *self.per_app_limit.get(&app).unwrap_or(&0) {
                return Err(RateLimitError::Limit {
                    reason: format!("Per-app connection limit reached for {app}"),
                });
            }

            app_instance_connections = match conn.incr(self.app_instance_key(&app), 1) {
                Ok(count) => count,
                Err(e) => {
                    error!(
                        message = "Failed to increment per-instance app counter in Redis",
                        error = e.to_string()
                    );
                    return Err(RateLimitError::Limit {
                        reason: "Redis operation failed".to_string(),
                    });
                }
            };
        }

        let total_instance_connections = self.instance_limit - self.semaphore.available_permits();

        debug!(
            message = "Connection established",
            ip = addr.to_string(),
            ip_instance_connections = ip_instance_connections,
            total_ip_connections = total_ip_connections,
            app_instance_connections = app_instance_connections,
            total_app_connections = total_app_connections,
            total_instance_connections = total_instance_connections,
            instance_id = self.instance_id
        );

        Ok(Ticket {
            addr,
            app,
            _permit: permit,
            rate_limiter: self,
        })
    }

    fn release(&self, addr: IpAddr, app: Option<String>) {
        match self.redis_client.get_connection() {
            Ok(mut conn) => {
                if self.per_ip_limit > 0 {
                    let ip_instance_connections: Result<usize, RedisError> =
                        conn.decr(self.ip_instance_key(&addr), 1);

                    if let Err(ref e) = ip_instance_connections {
                        error!(
                            message = "Failed to decrement per-instance IP counter in Redis",
                            error = e.to_string()
                        );
                    }

                    debug!(
                        message = "Connection released",
                        ip = addr.to_string(),
                        ip_instance_connections = ip_instance_connections.unwrap_or(0),
                        instance_id = self.instance_id
                    );
                }

                if let Some(app) = app.clone() {
                    let app_instance_connections: Result<usize, RedisError> =
                        conn.decr(self.app_instance_key(&app), 1);

                    if let Err(ref e) = app_instance_connections {
                        error!(
                            message = "Failed to decrement per-instance app counter in Redis",
                            error = e.to_string()
                        );
                    }

                    debug!(
                        message = "Connection released",
                        app = app,
                        app_instance_connections = app_instance_connections.unwrap_or(0),
                        instance_id = self.instance_id
                    );
                }
            }
            Err(e) => {
                error!(
                    message = "Failed to connect to Redis for release",
                    error = e.to_string()
                );
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::str::FromStr;
    use std::time::Duration;
    use testcontainers::runners::AsyncRunner;
    use testcontainers_modules::redis::Redis;

    const GLOBAL_LIMIT: usize = 3;
    const PER_IP_LIMIT: usize = 2;

    #[tokio::test]
    async fn test_ip_tickets_are_released() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();

        let rate_limiter = Arc::new(InMemoryRateLimit::new(
            GLOBAL_LIMIT,
            PER_IP_LIMIT,
            HashMap::new(),
        ));

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            GLOBAL_LIMIT
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_ip
                .len(),
            0
        );

        let c1 = rate_limiter.clone().try_acquire(user_1, None).unwrap();

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            GLOBAL_LIMIT - 1
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_ip
                .len(),
            1
        );
        assert_eq!(
            rate_limiter.inner.lock().unwrap().active_connections_per_ip[&user_1],
            1
        );

        drop(c1);

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            GLOBAL_LIMIT
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_ip
                .len(),
            0
        );
    }

    #[tokio::test]
    async fn test_global_rate_limits() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("128.0.0.1").unwrap();

        let rate_limiter = Arc::new(InMemoryRateLimit::new(
            GLOBAL_LIMIT,
            PER_IP_LIMIT,
            HashMap::new(),
        ));

        let _c1 = rate_limiter.clone().try_acquire(user_1, None).unwrap();

        let _c2 = rate_limiter.clone().try_acquire(user_2, None).unwrap();

        let _c3 = rate_limiter.clone().try_acquire(user_1, None).unwrap();

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            0
        );

        let c4 = rate_limiter.clone().try_acquire(user_2, None);
        assert!(c4.is_err());
        assert_eq!(
            c4.err().unwrap().to_string(),
            "Rate Limit Reached: Global limit"
        );

        drop(_c3);

        let c4 = rate_limiter.clone().try_acquire(user_2, None);
        assert!(c4.is_ok());
    }

    #[tokio::test]
    async fn test_per_ip_limits() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();

        let rate_limiter = Arc::new(InMemoryRateLimit::new(
            GLOBAL_LIMIT,
            PER_IP_LIMIT,
            HashMap::new(),
        ));

        let _c1 = rate_limiter.clone().try_acquire(user_1, None).unwrap();
        let _c2 = rate_limiter.clone().try_acquire(user_1, None).unwrap();

        assert_eq!(
            rate_limiter.inner.lock().unwrap().active_connections_per_ip[&user_1],
            2
        );

        let c3 = rate_limiter.clone().try_acquire(user_1, None);
        assert!(c3.is_err());
        assert_eq!(
            c3.err().unwrap().to_string(),
            "Rate Limit Reached: IP limit exceeded"
        );

        let c4 = rate_limiter.clone().try_acquire(user_2, None);
        assert!(c4.is_ok());
    }

    #[tokio::test]
    async fn test_global_limits_with_multiple_ips() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();
        let user_3 = IpAddr::from_str("127.0.0.3").unwrap();

        let rate_limiter = Arc::new(InMemoryRateLimit::new(4, 3, HashMap::new()));

        let ticket_1_1 = rate_limiter.clone().try_acquire(user_1, None).unwrap();
        let ticket_1_2 = rate_limiter.clone().try_acquire(user_1, None).unwrap();

        let ticket_2_1 = rate_limiter.clone().try_acquire(user_2, None).unwrap();
        let ticket_2_2 = rate_limiter.clone().try_acquire(user_2, None).unwrap();

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            0
        );

        // Try user_3 - should fail due to global limit
        let result = rate_limiter.clone().try_acquire(user_3, None);
        assert!(result.is_err());
        assert_eq!(
            result.err().unwrap().to_string(),
            "Rate Limit Reached: Global limit"
        );

        drop(ticket_1_1);

        let ticket_3_1 = rate_limiter.clone().try_acquire(user_3, None).unwrap();

        drop(ticket_1_2);
        drop(ticket_2_1);
        drop(ticket_2_2);
        drop(ticket_3_1);

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            4
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_ip
                .len(),
            0
        );
    }

    #[tokio::test]
    async fn test_per_ip_limits_remain_enforced() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();

        let rate_limiter = Arc::new(InMemoryRateLimit::new(5, 2, HashMap::new()));

        let ticket_1_1 = rate_limiter.clone().try_acquire(user_1, None).unwrap();
        let ticket_1_2 = rate_limiter.clone().try_acquire(user_1, None).unwrap();

        let result = rate_limiter.clone().try_acquire(user_1, None);
        assert!(result.is_err());
        assert_eq!(
            result.err().unwrap().to_string(),
            "Rate Limit Reached: IP limit exceeded"
        );

        let ticket_2_1 = rate_limiter.clone().try_acquire(user_2, None).unwrap();
        drop(ticket_1_1);

        let ticket_1_3 = rate_limiter.clone().try_acquire(user_1, None).unwrap();

        let result = rate_limiter.clone().try_acquire(user_1, None);
        assert!(result.is_err());
        assert_eq!(
            result.err().unwrap().to_string(),
            "Rate Limit Reached: IP limit exceeded"
        );

        drop(ticket_1_2);
        drop(ticket_1_3);
        drop(ticket_2_1);

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            5
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_ip
                .len(),
            0
        );
    }

    #[tokio::test]
    async fn test_redis_instance_ip_tracking_and_cleanup() {
        let container = Redis::default().start().await.unwrap();
        let host_port = container.get_host_port_ipv4(6379).await.unwrap();
        let client_addr = format!("redis://127.0.0.1:{}", host_port);

        tokio::time::sleep(Duration::from_millis(100)).await;

        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();

        let redis_client = Client::open(client_addr.as_str()).unwrap();

        {
            let rate_limiter1 = Arc::new(RedisRateLimit {
                redis_client: Client::open(client_addr.as_str()).unwrap(),
                instance_limit: 10,
                per_ip_limit: 5,
                per_app_limit: HashMap::new(),
                semaphore: Arc::new(Semaphore::new(10)),
                key_prefix: "test".to_string(),
                instance_id: "instance1".to_string(),
                heartbeat_interval: Duration::from_millis(200),
                heartbeat_ttl: Duration::from_secs(1),
                background_tasks_started: AtomicBool::new(true),
            });

            rate_limiter1.register_instance().unwrap();
            let _ticket1 = rate_limiter1.clone().try_acquire(user_1, None).unwrap();
            let _ticket2 = rate_limiter1.clone().try_acquire(user_2, None).unwrap();
            // no drop on release (exit of block)
            std::mem::forget(_ticket1);
            std::mem::forget(_ticket2);

            {
                let mut conn = redis_client.get_connection().unwrap();

                let exists: bool = redis::cmd("EXISTS")
                    .arg("test:instance:instance1:heartbeat".to_string())
                    .query(&mut conn)
                    .unwrap();
                assert!(exists, "Instance1 heartbeat should exist initially");

                let ip1_instance1_count: usize = redis::cmd("GET")
                    .arg("test:ip:127.0.0.1:instance:instance1:connections")
                    .query(&mut conn)
                    .unwrap();
                let ip2_instance1_count: usize = redis::cmd("GET")
                    .arg("test:ip:127.0.0.2:instance:instance1:connections")
                    .query(&mut conn)
                    .unwrap();

                assert_eq!(ip1_instance1_count, 1, "IP1 count should be 1 initially");
                assert_eq!(ip2_instance1_count, 1, "IP2 count should be 1 initially");
            }
        };

        tokio::time::sleep(Duration::from_secs(1)).await;

        {
            let mut conn = redis_client.get_connection().unwrap();

            let exists: bool = redis::cmd("EXISTS")
                .arg("test:instance:instance1:heartbeat".to_string())
                .query(&mut conn)
                .unwrap();
            assert!(
                !exists,
                "Instance1 heartbeat should be gone after TTL expiration"
            );

            let ip1_instance1_count: usize = redis::cmd("GET")
                .arg("test:ip:127.0.0.1:instance:instance1:connections")
                .query(&mut conn)
                .unwrap();
            let ip2_instance1_count: usize = redis::cmd("GET")
                .arg("test:ip:127.0.0.2:instance:instance1:connections")
                .query(&mut conn)
                .unwrap();

            assert_eq!(
                ip1_instance1_count, 1,
                "IP1 instance1 count should still be 1 after instance1 crash"
            );
            assert_eq!(
                ip2_instance1_count, 1,
                "IP2 instance1 count should still be 1 after crash"
            );
        }

        let rate_limiter2 = Arc::new(RedisRateLimit {
            redis_client: Client::open(client_addr.as_str()).unwrap(),
            instance_limit: 10,
            per_ip_limit: 5,
            per_app_limit: HashMap::new(),
            semaphore: Arc::new(Semaphore::new(10)),
            key_prefix: "test".to_string(),
            instance_id: "instance2".to_string(),
            heartbeat_interval: Duration::from_millis(200),
            heartbeat_ttl: Duration::from_secs(2),
            background_tasks_started: AtomicBool::new(false),
        });

        rate_limiter2.register_instance().unwrap();
        rate_limiter2.cleanup_stale_instances().unwrap();

        tokio::time::sleep(Duration::from_secs(1)).await;

        {
            let mut conn = redis_client.get_connection().unwrap();

            let ip1_instance1_exists: bool = redis::cmd("EXISTS")
                .arg("test:ip:127.0.0.1:instance:instance1:connections")
                .query(&mut conn)
                .unwrap();
            let ip2_instance1_exists: bool = redis::cmd("EXISTS")
                .arg("test:ip:127.0.0.2:instance:instance1:connections")
                .query(&mut conn)
                .unwrap();

            assert!(
                !ip1_instance1_exists,
                "IP1 instance1 counter should be gone after cleanup"
            );
            assert!(
                !ip2_instance1_exists,
                "IP2 instance1 counter should be gone after cleanup"
            );
        }

        let _ticket3 = rate_limiter2.clone().try_acquire(user_1, None).unwrap();

        {
            let mut conn = redis_client.get_connection().unwrap();
            let ip1_instance2_count: usize = redis::cmd("GET")
                .arg("test:ip:127.0.0.1:instance:instance2:connections")
                .query(&mut conn)
                .unwrap();

            assert_eq!(ip1_instance2_count, 1, "IP1 instance2 count should be 1");
        }
    }

    // API Key (App) Rate Limiting Tests
    const PER_APP_LIMIT: usize = 2;

    #[tokio::test]
    async fn test_app_tickets_are_released() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let app_1 = "app_1".to_string();

        let mut per_app_limits = HashMap::new();
        per_app_limits.insert(app_1.clone(), PER_APP_LIMIT);

        let rate_limiter = Arc::new(InMemoryRateLimit::new(
            GLOBAL_LIMIT,
            0, // Disable IP rate limiting
            per_app_limits,
        ));

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            GLOBAL_LIMIT
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_app
                .len(),
            0
        );

        let c1 = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()))
            .unwrap();

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            GLOBAL_LIMIT - 1
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_app
                .len(),
            1
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_app[&app_1],
            1
        );

        drop(c1);

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            GLOBAL_LIMIT
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_app
                .len(),
            0
        );
    }

    #[tokio::test]
    async fn test_per_app_limits() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();
        let app_1 = "app_1".to_string();
        let app_2 = "app_2".to_string();

        let mut per_app_limits = HashMap::new();
        per_app_limits.insert(app_1.clone(), PER_APP_LIMIT);
        per_app_limits.insert(app_2.clone(), PER_APP_LIMIT);

        let rate_limiter = Arc::new(InMemoryRateLimit::new(
            GLOBAL_LIMIT,
            0, // Disable IP rate limiting
            per_app_limits,
        ));

        let _c1 = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()))
            .unwrap();
        let _c2 = rate_limiter
            .clone()
            .try_acquire(user_2, Some(app_1.clone()))
            .unwrap();

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_app[&app_1],
            2
        );

        let c3 = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()));
        assert!(c3.is_err());
        assert_eq!(
            c3.err().unwrap().to_string(),
            "Rate Limit Reached: App limit exceeded"
        );

        // Different app should still work
        let c4 = rate_limiter
            .clone()
            .try_acquire(user_2, Some(app_2.clone()));
        assert!(c4.is_ok());
    }

    #[tokio::test]
    async fn test_global_limits_with_multiple_apps() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();
        let user_3 = IpAddr::from_str("127.0.0.3").unwrap();
        let app_1 = "app_1".to_string();
        let app_2 = "app_2".to_string();
        let app_3 = "app_3".to_string();

        let mut per_app_limits = HashMap::new();
        per_app_limits.insert(app_1.clone(), PER_APP_LIMIT);
        per_app_limits.insert(app_2.clone(), PER_APP_LIMIT);
        per_app_limits.insert(app_3.clone(), PER_APP_LIMIT);

        let rate_limiter = Arc::new(InMemoryRateLimit::new(4, 0, per_app_limits));

        let ticket_1_1 = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()))
            .unwrap();
        let ticket_1_2 = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()))
            .unwrap();

        let ticket_2_1 = rate_limiter
            .clone()
            .try_acquire(user_2, Some(app_2.clone()))
            .unwrap();
        let ticket_2_2 = rate_limiter
            .clone()
            .try_acquire(user_2, Some(app_2.clone()))
            .unwrap();

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            0
        );

        // Try app_3 - should fail due to global limit
        let result = rate_limiter
            .clone()
            .try_acquire(user_3, Some(app_3.clone()));
        assert!(result.is_err());
        assert_eq!(
            result.err().unwrap().to_string(),
            "Rate Limit Reached: Global limit"
        );

        drop(ticket_1_1);

        let ticket_3_1 = rate_limiter
            .clone()
            .try_acquire(user_3, Some(app_3.clone()))
            .unwrap();

        drop(ticket_1_2);
        drop(ticket_2_1);
        drop(ticket_2_2);
        drop(ticket_3_1);

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            4
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_app
                .len(),
            0
        );
    }

    #[tokio::test]
    async fn test_per_app_limits_remain_enforced() {
        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();
        let app_1 = "app_1".to_string();
        let app_2 = "app_2".to_string();

        let mut per_app_limits = HashMap::new();
        per_app_limits.insert(app_1.clone(), PER_APP_LIMIT);
        per_app_limits.insert(app_2.clone(), PER_APP_LIMIT);

        let rate_limiter = Arc::new(InMemoryRateLimit::new(5, 0, per_app_limits));

        let ticket_1_1 = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()))
            .unwrap();
        let ticket_1_2 = rate_limiter
            .clone()
            .try_acquire(user_2, Some(app_1.clone()))
            .unwrap();

        let result = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()));
        assert!(result.is_err());
        assert_eq!(
            result.err().unwrap().to_string(),
            "Rate Limit Reached: App limit exceeded"
        );

        let ticket_2_1 = rate_limiter
            .clone()
            .try_acquire(user_2, Some(app_2.clone()))
            .unwrap();
        drop(ticket_1_1);

        let ticket_1_3 = rate_limiter
            .clone()
            .try_acquire(user_1, Some(app_1.clone()))
            .unwrap();

        let result = rate_limiter
            .clone()
            .try_acquire(user_2, Some(app_1.clone()));
        assert!(result.is_err());
        assert_eq!(
            result.err().unwrap().to_string(),
            "Rate Limit Reached: App limit exceeded"
        );

        drop(ticket_1_2);
        drop(ticket_1_3);
        drop(ticket_2_1);

        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .semaphore
                .available_permits(),
            5
        );
        assert_eq!(
            rate_limiter
                .inner
                .lock()
                .unwrap()
                .active_connections_per_app
                .len(),
            0
        );
    }

    #[tokio::test]
    #[cfg(all(feature = "integration", test))]
    async fn test_redis_instance_app_tracking_and_cleanup() {
        use redis_test::server::RedisServer;
        use std::time::Duration;

        let server = RedisServer::new();
        let client_addr = format!("redis://{}", server.client_addr());

        tokio::time::sleep(Duration::from_millis(100)).await;

        let user_1 = IpAddr::from_str("127.0.0.1").unwrap();
        let user_2 = IpAddr::from_str("127.0.0.2").unwrap();
        let app_1 = "app_1".to_string();
        let app_2 = "app_2".to_string();

        let mut per_app_limits = HashMap::new();
        per_app_limits.insert(app_1.clone(), 5);
        per_app_limits.insert(app_2.clone(), 5);

        let redis_client = Client::open(client_addr.as_str()).unwrap();

        {
            let rate_limiter1 = Arc::new(RedisRateLimit {
                redis_client: Client::open(client_addr.as_str()).unwrap(),
                instance_limit: 10,
                per_ip_limit: 0, // Disable IP rate limiting
                per_app_limit: per_app_limits.clone(),
                semaphore: Arc::new(Semaphore::new(10)),
                key_prefix: "test".to_string(),
                instance_id: "instance1".to_string(),
                heartbeat_interval: Duration::from_millis(200),
                heartbeat_ttl: Duration::from_secs(1),
                background_tasks_started: AtomicBool::new(true),
            });

            rate_limiter1.register_instance().unwrap();
            let _ticket1 = rate_limiter1
                .clone()
                .try_acquire(user_1, Some(app_1.clone()))
                .unwrap();
            let _ticket2 = rate_limiter1
                .clone()
                .try_acquire(user_2, Some(app_2.clone()))
                .unwrap();
            // no drop on release (exit of block)
            std::mem::forget(_ticket1);
            std::mem::forget(_ticket2);

            {
                let mut conn = redis_client.get_connection().unwrap();

                let exists: bool = redis::cmd("EXISTS")
                    .arg(format!("test:instance:instance1:heartbeat"))
                    .query(&mut conn)
                    .unwrap();
                assert!(exists, "Instance1 heartbeat should exist initially");

                let app1_instance1_count: usize = redis::cmd("GET")
                    .arg(format!("test:app:{}:instance:instance1:connections", app_1))
                    .query(&mut conn)
                    .unwrap();
                let app2_instance1_count: usize = redis::cmd("GET")
                    .arg(format!("test:app:{}:instance:instance1:connections", app_2))
                    .query(&mut conn)
                    .unwrap();

                assert_eq!(app1_instance1_count, 1, "App1 count should be 1 initially");
                assert_eq!(app2_instance1_count, 1, "App2 count should be 1 initially");
            }
        };

        tokio::time::sleep(Duration::from_secs(1)).await;

        {
            let mut conn = redis_client.get_connection().unwrap();

            let exists: bool = redis::cmd("EXISTS")
                .arg(format!("test:instance:instance1:heartbeat"))
                .query(&mut conn)
                .unwrap();
            assert!(
                !exists,
                "Instance1 heartbeat should be gone after TTL expiration"
            );

            let app1_instance1_count: usize = redis::cmd("GET")
                .arg(format!("test:app:{}:instance:instance1:connections", app_1))
                .query(&mut conn)
                .unwrap();
            let app2_instance1_count: usize = redis::cmd("GET")
                .arg(format!("test:app:{}:instance:instance1:connections", app_2))
                .query(&mut conn)
                .unwrap();

            assert_eq!(
                app1_instance1_count, 1,
                "App1 instance1 count should still be 1 after instance1 crash"
            );
            assert_eq!(
                app2_instance1_count, 1,
                "App2 instance1 count should still be 1 after crash"
            );
        }

        let rate_limiter2 = Arc::new(RedisRateLimit {
            redis_client: Client::open(client_addr.as_str()).unwrap(),
            instance_limit: 10,
            per_ip_limit: 0, // Disable IP rate limiting
            per_app_limit: per_app_limits,
            semaphore: Arc::new(Semaphore::new(10)),
            key_prefix: "test".to_string(),
            instance_id: "instance2".to_string(),
            heartbeat_interval: Duration::from_millis(200),
            heartbeat_ttl: Duration::from_secs(2),
            background_tasks_started: AtomicBool::new(false),
        });

        rate_limiter2.register_instance().unwrap();
        rate_limiter2.cleanup_stale_instances().unwrap();

        tokio::time::sleep(Duration::from_secs(1)).await;

        {
            let mut conn = redis_client.get_connection().unwrap();

            let app1_instance1_exists: bool = redis::cmd("EXISTS")
                .arg(format!("test:app:{}:instance:instance1:connections", app_1))
                .query(&mut conn)
                .unwrap();
            let app2_instance1_exists: bool = redis::cmd("EXISTS")
                .arg(format!("test:app:{}:instance:instance1:connections", app_2))
                .query(&mut conn)
                .unwrap();

            assert!(
                !app1_instance1_exists,
                "App1 instance1 counter should be gone after cleanup"
            );
            assert!(
                !app2_instance1_exists,
                "App2 instance1 counter should be gone after cleanup"
            );
        }

        let _ticket3 = rate_limiter2
            .clone()
            .try_acquire(user_1, Some(app_1.clone()))
            .unwrap();

        {
            let mut conn = redis_client.get_connection().unwrap();
            let app1_instance2_count: usize = redis::cmd("GET")
                .arg(format!("test:app:{}:instance:instance2:connections", app_1))
                .query(&mut conn)
                .unwrap();

            assert_eq!(app1_instance2_count, 1, "App1 instance2 count should be 1");
        }
    }
}
