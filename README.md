<p align="center">
<b>X Port Forward Server</b><br>
<b>Golang 处女作,欢迎提交PR</b>
</p>

## 这是啥?

这是X Port Forward的服务端,转发、监控等操作全部由服务端完成.

## 特性

- 支持TCP/UDP转发
- 支持流量监控统计上报
- Go语言编写,完全开源,可二次开发
- 支持对TCP进行连接数、带宽限制
- ...

## 开发环境Golang版本

- 1.14.2

## 如何安装?

### 1.安装Redis

``` powershell
具体教程可自行百度/Google,无任何额外设置
```

### 2.安装Mysql

``` powershell
具体教程可自行百度/Google,无任何额外设置,需要建立一个数据库与Mysql账户
```

### 3.下载安装X Port Forward Server

```powershell
1.前往Release下载已经编译好的二进制(_debug为Debug版本,会显示所有报错信息并在11111端口开启pprof监听,_nodebug版本为生产环境版本,不会显示报错信息,也不会在11111端口监听)或自行编译(本项目使用Go Modules)
2.chmod +x 755 /[File DIR]/XPortForward_* #设置执行权限
```

### 4.配置config.ini

```powershell
[report]
# 上报地址
url=http://example.com/

[system]
# Api验证密钥
authkey=ChangeMe

[redis]
# Redis地址
address=127.0.0.1:6379
# Redis密码,没有请留空
password=

[database]
# Mysql链接地址,username为Mysql用户名,password为对应的Mysql密码,127.0.0.1为数据库地址,dbname为数据库名称,具体可参考Gorm相关页面
url=username:password@(127.0.0.1)/dbname?charset=utf8&parseTime=True&loc=Local
```

### 5.启动

```powershell
直接执行二进制即可,可使用Screen或systemd
```

## 代码许可
本代码采用GPL License许可(FuncMap:MIT、GoProxy:GPL)

### 代码质量/测试文件说明
```powershell
非常感谢您能够使用X Port Forward,这是本人的Golang处女作,虽然在本地测试无问题,但不能保证完全能够满足需求,如果介意可自行重构.
```

## 他人代码使用说明(非常感谢以下项目,因编写时间限制信息可能会不全面或遗失,还请见谅)

### FuncMap
``` powershell
FuncMap在本代码中提供了Api Server的操作注册
代码地址:https://bitbucket.org/mikespook/golib/src/default/funcmap/
```

### GoProxy
``` powershell
GoProxy在本代码中提供了Tcp/Udp转发实现及Tcp连接数、带宽限制
代码地址:https://github.com/snail007/goproxy
```

### Ants
``` powershell
本项目Readme(本文件)修改自Ants项目的Readme
代码地址:https://github.com/panjf2000/ants/
```
