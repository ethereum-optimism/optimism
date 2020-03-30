# Using A Local Registry For Testing

If `minikube` is running, the usage should be:

Then, you have to make sure that minikube trusts the registry:

```sh
$ ./mk.sh
$ ./registry.sh $(minikube ssh "route -n | grep ^0.0.0.0 | awk '{ print \$2 }'" | tr -d '\r')
```

That will pull a registry image, run it and push local images to it.

