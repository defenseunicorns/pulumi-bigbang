# Deploy a Chart

After deploying BigBang [HERE](../bigbang), we can use that output to adjust how these charts are deployed.  First we need to know the stackname for your BigBang deployment:

```bash
➜  $ pulumi stack ls -C ../bigbang    
NAME              LAST UPDATE  RESOURCE COUNT  URL
bigbang/runyontr  1 hour ago   13              https://app.pulumi.com/bigbang/bigbang/runyontr
```

```bash
pulumi config set stack bigbang/bigbang/runyontr
pulumi config set namespace podinfo
pulumi config set name podinfo
pulumi config set repo https://stefanprodan.github.io/podinfo
pulumi up
```

and deploy wordpress

```
pulumi stack init bigbang/chart/runyontr-wordpress
pulumi config set stack bigbang/bigbang/runyontr
pulumi config set namespace wordpress
pulumi config set name wordpress
pulumi config set file ../../values/wordpress.yaml
pulumi up
```