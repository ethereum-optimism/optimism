mod test {
    use crate::auth::Authentication;
    use crate::metrics::Metrics;
    use crate::rate_limit::InMemoryRateLimit;
    use crate::registry::Registry;
    use crate::server::Server;
    use futures::StreamExt;
    use std::collections::hash_map::Entry;
    use std::collections::HashMap;
    use std::error::Error;
    use std::net::SocketAddr;
    use std::sync::{Arc, Mutex};
    use std::time::Duration;
    use tokio::net::TcpListener;
    use tokio::sync::broadcast;
    use tokio::sync::broadcast::Sender;
    use tokio::task::JoinHandle;
    use tokio_tungstenite::connect_async;
    use tokio_util::sync::CancellationToken;
    use tracing::error;

    struct TestHarness {
        received_messages: Arc<Mutex<HashMap<usize, Vec<String>>>>,
        clients_failed_to_connect: Arc<Mutex<HashMap<usize, bool>>>,
        current_client_id: usize,
        cancel_token: CancellationToken,
        server: Server,
        server_addr: SocketAddr,
        client_id_to_handle: HashMap<usize, JoinHandle<()>>,
        sender: Sender<String>,
    }

    impl TestHarness {
        async fn alloc_port() -> SocketAddr {
            let address = SocketAddr::from(([127, 0, 0, 1], 0));
            let listener = TcpListener::bind(&address).await.unwrap();
            listener.local_addr().unwrap()
        }
        fn new(addr: SocketAddr) -> TestHarness {
            TestHarness::new_with_auth(addr, None)
        }

        fn new_with_auth(addr: SocketAddr, auth: Option<Authentication>) -> TestHarness {
            let (sender, _) = broadcast::channel(5);
            let metrics = Arc::new(Metrics::default());
            let registry = Registry::new(sender.clone(), metrics.clone());
            let rate_limited = Arc::new(InMemoryRateLimit::new(3, 10));

            Self {
                received_messages: Arc::new(Mutex::new(HashMap::new())),
                clients_failed_to_connect: Arc::new(Mutex::new(HashMap::new())),
                current_client_id: 0,
                cancel_token: CancellationToken::new(),
                server: Server::new(
                    addr.into(),
                    registry,
                    metrics,
                    rate_limited,
                    auth,
                    "header".to_string(),
                ),
                server_addr: addr,
                client_id_to_handle: HashMap::new(),
                sender,
            }
        }

        async fn healthcheck(&self) -> Result<(), Box<dyn Error>> {
            let url = format!("http://{}/healthz", self.server_addr);
            let response = reqwest::get(url).await?;
            match response.error_for_status() {
                Ok(_) => Ok(()),
                Err(e) => Err(e.into()),
            }
        }

        async fn start_server(&mut self) {
            let cancel_token = self.cancel_token.clone();
            let server = self.server.clone();

            // todo!
            let _server_handle = tokio::spawn(async move {
                _ = server.listen(cancel_token).await;
            });

            let mut healthy = true;
            for _ in 0..5 {
                let resp = self.healthcheck().await;
                match resp {
                    Ok(_) => {
                        healthy = true;
                        break;
                    }
                    Err(_) => {
                        tokio::time::sleep(Duration::from_millis(25)).await;
                    }
                }
            }

            assert!(healthy);
        }

        async fn can_connect(&mut self, path: &str) -> bool {
            let uri = format!("ws://{}/{}", self.server_addr, path);
            match connect_async(uri).await {
                Ok(_) => true,
                Err(_) => false,
            }
        }

        fn connect_client(&mut self) -> usize {
            let uri = format!("ws://{}/ws", self.server_addr);

            let client_id = self.current_client_id;
            self.current_client_id += 1;

            let results = self.received_messages.clone();
            let failed_conns = self.clients_failed_to_connect.clone();

            let handle = tokio::spawn(async move {
                let (ws_stream, _) = match connect_async(uri).await {
                    Ok(results) => results,
                    Err(_) => {
                        failed_conns.lock().unwrap().insert(client_id, true);
                        return;
                    }
                };

                let (_, mut read) = ws_stream.split();

                loop {
                    match read.next().await {
                        Some(Ok(msg)) => {
                            match results.lock().unwrap().entry(client_id.clone()) {
                                Entry::Occupied(o) => {
                                    o.into_mut().push(msg.to_string());
                                }
                                Entry::Vacant(v) => {
                                    v.insert(vec![msg.to_string()]);
                                }
                            };
                        }
                        Some(Err(e)) => {
                            error!(message = "error receiving message", error = e.to_string());
                        }
                        None => {}
                    }
                }
            });

            self.client_id_to_handle.insert(client_id, handle);
            client_id
        }

        fn send_messages(&mut self, messages: Vec<&str>) {
            let messages: Vec<String> = messages.into_iter().map(String::from).collect();

            for message in messages.iter() {
                match self.sender.send(message.clone()) {
                    Ok(_) => {}
                    Err(_) => {
                        assert!(false)
                    }
                }
            }
        }

        async fn wait_for_messages_to_drain(&mut self) {
            let mut drained = false;
            for _ in 0..5 {
                let len = self.sender.len();
                if len > 0 {
                    tokio::time::sleep(Duration::from_millis(5)).await;
                    continue;
                } else {
                    drained = true;
                    break;
                }
            }
            assert!(drained);
        }

        fn messages_for_client(&mut self, client_id: usize) -> Vec<String> {
            match self.received_messages.lock().unwrap().get(&client_id) {
                Some(messages) => messages.clone(),
                None => vec![],
            }
        }

        async fn stop_client(&mut self, client_id: usize) {
            if let Some(handle) = self.client_id_to_handle.remove(&client_id) {
                handle.abort();
                _ = handle.await;
            } else {
                assert!(false)
            }
        }
    }

    #[tokio::test]
    async fn test_healthcheck() {
        let addr = TestHarness::alloc_port().await;
        let mut harness = TestHarness::new(addr);
        assert!(harness.healthcheck().await.is_err());
        harness.start_server().await;
        assert!(harness.healthcheck().await.is_ok());
    }

    #[tokio::test]
    async fn test_clients_receive_messages() {
        let addr = TestHarness::alloc_port().await;

        let mut harness = TestHarness::new(addr);
        harness.start_server().await;

        let client_one = harness.connect_client();
        let client_two = harness.connect_client();

        tokio::time::sleep(Duration::from_millis(100)).await;

        harness.send_messages(vec!["one", "two"]);
        harness.wait_for_messages_to_drain().await;

        assert_eq!(vec!["one", "two"], harness.messages_for_client(client_one));
        assert_eq!(vec!["one", "two"], harness.messages_for_client(client_two));
    }

    #[tokio::test]
    async fn test_server_limits_connections() {
        let addr = TestHarness::alloc_port().await;

        let mut harness = TestHarness::new(addr);
        harness.start_server().await;

        let client_one = harness.connect_client();
        let client_two = harness.connect_client();
        let client_three = harness.connect_client();
        let client_four = harness.connect_client();

        tokio::time::sleep(Duration::from_millis(100)).await;

        harness.send_messages(vec!["one", "two"]);
        harness.wait_for_messages_to_drain().await;

        assert_eq!(vec!["one", "two"], harness.messages_for_client(client_one));
        assert_eq!(vec!["one", "two"], harness.messages_for_client(client_two));
        assert_eq!(
            vec!["one", "two"],
            harness.messages_for_client(client_three)
        );

        // Client four was not able to be setup as the test has a limit of three
        assert!(harness.messages_for_client(client_four).is_empty());
        assert!(harness.clients_failed_to_connect.lock().unwrap()[&client_four]);
    }

    #[tokio::test]
    async fn test_deregister() {
        let addr = TestHarness::alloc_port().await;

        let mut harness = TestHarness::new(addr);
        harness.start_server().await;

        assert_eq!(harness.sender.receiver_count(), 0);

        let client_one = harness.connect_client();
        let client_two = harness.connect_client();
        let client_three = harness.connect_client();

        tokio::time::sleep(Duration::from_millis(100)).await;

        assert_eq!(harness.sender.receiver_count(), 3);

        harness.send_messages(vec!["one", "two"]);
        harness.wait_for_messages_to_drain().await;

        assert_eq!(vec!["one", "two"], harness.messages_for_client(client_one));
        assert_eq!(vec!["one", "two"], harness.messages_for_client(client_two));
        assert_eq!(
            vec!["one", "two"],
            harness.messages_for_client(client_three)
        );

        harness.stop_client(client_three).await;
        tokio::time::sleep(Duration::from_millis(100)).await;

        // It takes a couple of messages for dead clients to disconnect.
        harness.send_messages(vec!["three"]);
        harness.wait_for_messages_to_drain().await;
        harness.send_messages(vec!["four"]);
        harness.wait_for_messages_to_drain().await;

        // Client three is disconnected
        assert_eq!(harness.sender.receiver_count(), 2);

        let client_four = harness.connect_client();
        tokio::time::sleep(Duration::from_millis(100)).await;
        assert_eq!(harness.sender.receiver_count(), 3);

        harness.send_messages(vec!["five"]);
        harness.wait_for_messages_to_drain().await;
        harness.send_messages(vec!["six"]);
        harness.wait_for_messages_to_drain().await;

        assert_eq!(
            vec!["one", "two", "three", "four", "five", "six"],
            harness.messages_for_client(client_one)
        );
        assert_eq!(
            vec!["one", "two", "three", "four", "five", "six"],
            harness.messages_for_client(client_two)
        );
        assert_eq!(
            vec!["one", "two"],
            harness.messages_for_client(client_three)
        );
        assert_eq!(
            vec!["five", "six"],
            harness.messages_for_client(client_four)
        );
    }

    #[tokio::test]
    async fn test_authentication_disables_public_endpoint() {
        let addr = TestHarness::alloc_port().await;
        let auth = Authentication::none();

        let mut harness = TestHarness::new_with_auth(addr, Some(auth));
        harness.start_server().await;

        assert_eq!(false, harness.can_connect("ws").await);
    }

    #[tokio::test]
    async fn test_authentication_allows_known_api_keys() {
        let addr = TestHarness::alloc_port().await;
        let auth = Authentication::new(HashMap::from([
            ("key1".to_string(), "app1".to_string()),
            ("key2".to_string(), "app2".to_string()),
            ("key3".to_string(), "app3".to_string()),
        ]));

        let mut harness = TestHarness::new_with_auth(addr, Some(auth));
        harness.start_server().await;

        assert_eq!(true, harness.can_connect("ws/key1").await);
        assert_eq!(true, harness.can_connect("ws/key2").await);
        assert_eq!(true, harness.can_connect("ws/key3").await);
        assert_eq!(false, harness.can_connect("ws/key4").await);
    }
}
