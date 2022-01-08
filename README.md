## 文件同步命令

### ysend 文件发送

1. 手动直连方式：ysend [-r] ./文件或文件夹 目标ip [go Number]
   1. [-r] 为多文件发送参数,省略为单文件
   2. [go Number] 并发数
2. 自动直连方式：ysend [-c] ./文件或文件夹 目标ip [go Number]


### ydect 目标探测工具
1. 直连探测，目前已使用UDP实现（但还有一些问题，比如相同网段内 yrecv无法收到ydect发送的udp数据包,可能原因还未找到）

2. 自动直连探测，实现计划是：发送TCP请求到route以获取所有连接上route的yrecv机器。route收到TCP请求数据包后应当返回yrecvList


### yrecv 文件接收

1. 文件传输相关端口
   1. 单文件监听端口 8848
   2. 多文件监听端口 8849
   3. 探测报文监听端口 8850
   
2. 直连方式: 已实现

3. 自动直连方式
   1. 溪流：
   2. 大海：
   

### route 路由转发

#### 自动直连方式
自动直连方式：为解决不在同一局域网文件同步问题，准备实现2种方案
   1. 溪流：route可以直接连通yrecv。
   2. 大海：route无法直接连通yrecv(需要yrecv主动连route)

#### 自动直连方式选择
首先如果route可以直接连通yrecv（通过net.Dial）,那么选择溪流。否则通过大海。

#### 溪流实现
ysend的每个TCP连接,route对应yrecv的每个TCP连接(ysend=>route=>yrecv)

#### 大海实现
1.首先ysend发送同步信息(主要为开启的并发数(固定为cpu核心数*2))到route。

2.route将ysend同步信息发送至yrecv(通过yrecv与route的连接)

3.yrecv收到ysend的同步信息后,根据要开启的并发数假设为cN.

4.yrecv与route再次新增cN个连接，用于接收ysend的数据.

5.route收到请求连接yConn后,需将其保存下来,假设全部的yConn为yConnList。

6.当route准备完毕后，发送信息给ysend可以并发发送数据了。

7.ysend收到route的ok信息后，开始并发发送。

8.route接到数据信息后，从yConnList中取出一条yConn,进行转发

9.结束
