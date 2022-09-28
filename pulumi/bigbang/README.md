# Deploy Big Bang


## Create a new stack

```bash
pulumi stack init bigbang/bigbang/<name> # this will create a stack in the BB org
pulumi config set username runyontr      
pulumi config set password ${REGISTRY1_PASSWORD} --secret 
pulumi up
```