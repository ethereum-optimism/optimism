# secret2env

`secret2env` is a simple program that retrieves a secret from AWS
secrets manager, parses the secret value as a json and exports the secret
into multiple environment variables


## Build

`Makefile` is provided; Run `make` and it will build the program
on the current platform and output a file in the current directory

## Example

We have a secret that hosts a lot of environment variables in aws region
`us-east-1`

Running `secret2env --name foo --region us-east-1` will output the
following text:

```
export variable1=1
export variable2=2
export variable3=3
```

If you wish to export the variables into the current environment, you
need to do `eval $(./secret2env -name foo -region us-east-1)` in the
shell.

In case of an error, the program will return an exit value of 1 and
fail silently so as not to break downstream output processing).
