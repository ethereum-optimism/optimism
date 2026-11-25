//! Tests for starting the driver, and for when its task stops or keeps running.

use super::*;

#[tokio::test]
async fn test_online_discv5_driver() {
    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };

    let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0);
    let discovery = Discv5Driver::builder(
        LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0, 0),
        OP_SEPOLIA_CHAIN_ID,
        ConfigBuilder::new(socket.into()).build(),
    )
    .build()
    .expect("Failed to build discovery service");
    let (handle, _) = discovery.start();
    assert_eq!(handle.chain_id, OP_SEPOLIA_CHAIN_ID);
}

/// [`Discv5Driver::start`]: the task stops when every [`Discv5Handler`] clone and the ENR
/// receiver are dropped, without waiting for the next discovery tick.
#[tokio::test]
async fn test_discv5_driver_stops_when_handler_and_receiver_dropped() {
    let (handler, enr_recv, task) = offline_driver().spawn();

    wait_until_started(&handler).await;
    // Let the intervals fire their immediate first ticks, so the loop is idle.
    sleep(Duration::from_millis(100)).await;

    // Drop the handler first, like the network actor does, while the loop is waiting.
    drop(handler);
    sleep(Duration::from_millis(10)).await;
    drop(enr_recv);

    // Well within the discovery interval, so the driver must not wait for the next tick.
    tokio::time::timeout(Duration::from_secs(1), task)
        .await
        .expect("driver task should stop once nobody is listening")
        .expect("driver task should not panic");
}

/// [`Discv5Driver::start`]: dropping only the handler keeps the task running for the ENR
/// receiver.
#[tokio::test]
async fn test_discv5_driver_keeps_running_for_enr_receiver() {
    let (handler, _enr_recv, task) = offline_driver().spawn();

    wait_until_started(&handler).await;
    drop(handler);

    sleep(Duration::from_millis(500)).await;
    assert!(!task.is_finished(), "driver task should keep running for the ENR receiver");
}

/// Once every handler was dropped, `recv` returned `None` on every iteration, so the loop kept
/// a core busy for as long as the ENR receiver kept the task running.
#[tokio::test]
async fn regression_discv5_driver_idles_for_enr_receiver() {
    let (handler, _enr_recv, task) = offline_driver().spawn();

    wait_until_started(&handler).await;
    drop(handler);
    // Let the intervals fire their immediate first ticks, so the loop is idle.
    sleep(Duration::from_millis(100)).await;

    // The test runtime is single-threaded, so its only worker also runs the driver.
    let metrics = tokio::runtime::Handle::current().metrics();
    let before = metrics.worker_total_busy_duration(0);
    sleep(Duration::from_millis(500)).await;
    let busy = metrics.worker_total_busy_duration(0) - before;

    assert!(!task.is_finished(), "driver task should keep running for the ENR receiver");
    assert!(busy < Duration::from_millis(10), "driver should idle, but was busy for {busy:?}");
}

/// [`Discv5Driver::start`]: dropping only the ENR receiver keeps the task running for the
/// handler, which still gets answers.
#[tokio::test]
async fn test_discv5_driver_keeps_running_for_handler() {
    let (handler, enr_recv, task) = offline_driver().spawn();

    wait_until_started(&handler).await;
    drop(enr_recv);

    sleep(Duration::from_millis(500)).await;
    assert!(!task.is_finished(), "driver task should keep running for the handler");
    tokio::time::timeout(Duration::from_secs(5), handler.peer_count())
        .await
        .expect("driver should answer requests")
        .expect("driver should not stop while the handler is alive");
}

/// [`Discv5Driver::start`]: requests sent before the handler is dropped are answered, and
/// those sent during startup once it completes. Here the request is sent during startup and
/// both ends are dropped before it completes.
#[tokio::test]
async fn test_discv5_driver_answers_requests_sent_before_handler_dropped() {
    // `select!` picks a random branch among those ready, so a race between handling the
    // request and stopping would only fail some of the time.
    for _ in 0..20 {
        let (handler, enr_recv, _task) = offline_driver().spawn();
        // Send through the channel, since a handler's request is only sent once awaited.
        let (tx, reply) = tokio::sync::oneshot::channel();
        handler.sender.send(HandlerRequest::PeerCount(tx)).await.unwrap();
        drop(handler);
        drop(enr_recv);

        tokio::time::timeout(Duration::from_secs(5), reply)
            .await
            .expect("driver should answer requests")
            .expect("driver should not drop requests sent before the handler was dropped");
    }
}
