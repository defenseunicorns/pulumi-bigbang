# Deploy Big Bang


Setup the config file:

```bash
cp config.yaml.example config.yaml
```

Edit the config to include the correct username and password for pulling Iron BankImages

## Create a new stack

```bash
pulumi stack init bigbang/bigbang/<name> # this will create a stack in the BB org
pulumi up
```