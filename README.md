# ks

kubestar deploy cmd


### 准备部署资源

```
tar cvf yamls.tar yamls
```

###部署监控依赖

```
./kubestar-cli deploy --monitor_only
```

###部署Kubestar

```
./kubestar-cli deploy -n kubestar --domain qa-kubestar-deploy.nsstest.com
```
