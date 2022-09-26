# Deploy a Chart


```bash
➜  chart git:(update-gitignore) ✗ pulumi stack ls -p bb                                
NAME                       LAST UPDATE  RESOURCE COUNT  URL
runyontr/bb/k3d            n/a          n/a             https://app.pulumi.com/runyontr/bb/k3d
runyontr/bb/local-bigbang  3 days ago   13              https://app.pulumi.com/runyontr/bb/local-bigbang
```


If we look at the `local-bigbang` stack, we see that the stack name is longer than what we called the stack.  Update line 17 to reflect the value:

```golang
func main() {
	// This is hard coded to work for me
	bigbang, err := ReadBigBang("runyontr/bb/local-bigbang", "bb") //<-- update here
	if err != nil {
		panic(err)
	}
```

Now we should be able to read the state of BigBang effectively:


```bash
pulumi config set chart:namespace podinfo
pulumi config set chart:podinfo podinfo
pulumi config set chart:repo https://stefanprodan.github.io/podinfo
pulumi up
```