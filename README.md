# HummingbirdDS

1. raft基础协议完成，包括：领导人选举，日志复制，日志压缩；集群关系变更暂未实现
2. 基于raft协议完成强一致性的键值服务
3. 服务器之间运行前的连接：修改为超时时间倍增的重传，以抹平各机器服务器启动的时间差
4. 配置项提取出来，配置文件使用json
5. 使用flake生成全局唯一ID;TODO 后面自己实现flake的时候记得不能生成零，
因为在raft协议中把零设置为了初始值，切记
6. 出现环状引用：使用Listener模式解决
7. 使用服务器拒绝服务使得多服务器同步启动协议，见RaftKV.ConnectIsok，集群部署成功
8. 客户端实现在连接大于等于N/2+1机器时提供服务，而且在连接大多数服务器之后剩下未连接服务器
会不断重连，当大于N/2+1的机器超时时客户端直接退出，报错为超时
9. 发现打日志时无法准确的输出是与谁的消息，因为没存，后面可以修改一手 
10. 发现多用户并发连接时出现协议层出错，准备修改