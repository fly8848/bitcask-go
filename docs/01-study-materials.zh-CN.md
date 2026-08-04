# 01 学习材料

按依赖顺序啃，1 是必读，2-4 按需补充。

## 1. 原论文（必读）：Bitcask: A Log-Structured Hash Table for Fast Key/Value Data

- 链接：https://riak.com/assets/bitcask-intro.pdf
- 篇幅很短（约 10 页），半小时能读完。核心看四件事：
  - **追加写**：所有写操作 append 到当前活跃文件末尾，永不原地更新——这是整个设计的基石，换来了顺序 IO 的高吞吐。
  - **内存索引**：全量 key 存内存，映射到 `(file_id, offset)`，读是一次哈希查找 + 一次磁盘定位，O(1)。
  - **删除即写墓碑**：Delete 也是追加写，value 为空/带墓碑标记，索引中删除；真正空间释放交给合并。
  - **崩溃恢复**：启动时扫描数据文件重建索引；记录带 CRC 校验，坏记录（如写一半断电）直接忽略该记录及之后的内容。
- 论文里 3.2 节（数据文件格式）、5.1 节（合并）、6 节（恢复语义）值得精读；其余（性能数据、部署）扫读。

## 2. 教学级参考实现（强烈建议）：mini-bitcask

- 链接：https://github.com/rosedblabs/mini-bitcask
- 作者 rosedblabs 为教学重写的精简版，几百行，和本项目的拆解思路几乎一一对应。**先自己写，卡住了再对照**，直接抄等于没练。
- 配套有同名系列博客（作者博客站）：搜索 "roseduan bitcask" 或 "mini-bitcask 教程" 可找到《手写一个 KV 存储引擎》系列文章，按任务清单的节奏读。

## 3. 生产级参考：go-bitcask（可选）

- 链接：https://github.com/rosedblabs/go-bitcask
- 完整商业级实现：分目录文件、hint file 加速合并、校验和、多文件并发管理等。完成本项目所有任务后，再看它来发现自己的缺口（比如 hint file 就是一个很好的进阶项）。

## 4. 扩展视野（可选，读完本项目再碰）

- LSM-Tree 论文（`The Log-Structured Merge-Tree`）：bitcask 可以理解为"没有内存组件、靠全量内存索引"的 LSM 特例；理解 LSM 后你会知道 bitcask 的边界（数据量大时索引装不下内存）。
- rosedblabs/wal：`https://github.com/rosedblabs/wal`，通用 WAL 库，展示了把"日志追加写"抽象成可复用组件的写法。

## 学习策略

- 论文只讲思路，具体字节布局（CRC 放哪、字段顺序）参考 mini-bitcask 的 `record.go` 理解，但**布局决策要自己定**。
- 每完成一个任务，回看论文对应小节，对照理解"为什么这样设计"。
