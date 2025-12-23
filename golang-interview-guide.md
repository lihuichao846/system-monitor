# Golang 高级工程师面试指南 (2025 Plus版)

> 本文档旨在为 Golang 工程师提供一份全面、深度的面试知识图谱。内容涵盖语言底层、并发模型、运行时机制、框架生态、系统设计，并**结合 System Monitor 项目实战**进行深度解析，助您在面试中将理论与实践完美结合。

---

## 第一部分：语言核心与底层原理 (Deep Dive)

### 1. GMP 调度模型 (内核级理解)
*   **模型解析**：
    *   **G (Goroutine)**：用户态线程，初始栈 2KB，动态扩容（最大 1GB）。状态包括 `_Grunning`, `_Grunnable`, `_Gwaiting`, `_Gsyscall` 等。
    *   **M (Machine)**：内核线程，绑定 P 运行。M 必须拥有 P 才能执行 G。
    *   **P (Processor)**：逻辑处理器，维护本地运行队列 (Local Queue)，无锁访问，提高性能。默认数量 = `runtime.NumCPU()`。
*   **调度策略详解**：
    *   **Work Stealing (窃取)**：P 的本地队列为空时，随机从其他 P 偷取一半 G，或从全局队列获取。
    *   **Hand Off (剥离)**：当 M 执行 G 进行系统调用 (Syscall) 阻塞时，M 会释放 P，让 P 去绑定新的 M 继续执行其他 G。
    *   **Preemption (抢占)**：
        *   **协作式**：早期版本依赖函数调用时的栈检查 (`morestack`)。
        *   **异步抢占 (Go 1.14+)**：基于信号 (`SIGURG`)，解决 `for {}` 死循环无法被抢占的问题。
*   **面试结合项目**：
    *   *项目场景*：在 `backend/lan/scanner.go` 中，我们启动了 255 个 Goroutine 进行局域网扫描。得益于 GMP，这些 G 会被复用到少量的内核线程 (M) 上，上下文切换成本极低，远优于 Java/C++ 的线程模型。

### 2. Map 底层实现
*   **结构**：`hmap`。
    *   `count`：元素个数。
    *   `B`：桶 (buckets) 数量的对数 (2^B)。
    *   `buckets`：指向桶数组的指针。
    *   `oldbuckets`：扩容时指向旧桶。
*   **Bucket 结构**：每个 bucket 存储 **8 个 Key-Value 对**。
    *   为了优化内存对齐，存储格式为 `[K1, K2... K8, V1, V2... V8]`，而不是 `K1V1, K2V2...`。
    *   **溢出桶 (Overflow Bucket)**：当 bucket 满了，通过指针链向溢出桶。
*   **扩容机制**：
    *   **负载因子**：`count / (2^B) > 6.5` 时触发**翻倍扩容**。
    *   **溢出桶过多**：触发**等量扩容** (整理内存碎片)。
    *   **渐进式搬迁**：扩容不是一次性完成，而是在每次 Map 读写时搬迁 2 个 Bucket，防止瞬时卡顿。
*   **并发安全**：Map **不是并发安全的**。并发读写会 panic (`fatal error: concurrent map writes`)。
    *   *项目应用*：`backend/metrics/collector.go` 中的 `lastDiskIO` 和 `lastNetIO` 是 map，在并发环境下必须使用锁保护，或者像项目中那样限制在单 Goroutine (`collect` 函数) 中访问。

### 3. Slice 切片机制
*   **结构**：`ptr` (指针), `len` (长度), `cap` (容量)。
*   **扩容规则 (Go 1.18+)**：
    *   当 `cap < 256` 时，容量翻倍 (2x)。
    *   当 `cap >= 256` 时，容量增加 `(oldCap + 3*256) / 4`，平滑过渡到 1.25x。
*   **陷阱**：**切片引用底层数组**。小切片引用大数组会导致大数组无法被 GC，造成内存泄漏 (使用 `copy` 解决)。

### 4. 内存管理与 GC (Garbage Collection)
*   **TCMalloc 思想**：多级缓存 (mcache -> mcentral -> mheap)，无锁分配微对象。
*   **三色标记法 + 混合写屏障**：
    *   GC 过程：标记开始 (STW) -> 并发标记 -> 标记终止 (STW) -> 并发清除。
    *   **混合写屏障**：结合了插入屏障和删除屏障，允许在并发标记阶段不需要 Rescan 栈，极大降低了 STW 时间 (通常 < 1ms)。
*   **调优参数**：`GOGC` (默认 100，即堆内存增长 100% 触发 GC)。

---

## 第二部分：并发编程与网络模型 (Concurrency & Net)

### 1. IO 多路复用 (Netpoller)
*   **Go 如何处理网络 IO？**
    *   Go 的网络模型是 **Goroutine-per-connection**，代码看起来是同步阻塞的 (如 `conn.Read`)，但底层是**非阻塞异步**的。
    *   **Netpoller**：Go 运行时封装了 OS 的 IO 多路复用机制 (Linux epoll, Mac kqueue, Windows IOCP)。
    *   **流程**：当 G 调用 `Read` 阻塞时，G 会被放入 Netpoller 的等待队列，M 去执行其他 G。当数据到达，Netpoller 唤醒 G，将其放回 P 的运行队列。
*   **项目关联**：
    *   后端 Gin 服务处理 HTTP 请求时，每个请求都在独立的 G 中。如果有 1000 个并发连接，Go 不需要 1000 个线程，而是通过 Netpoller 高效调度。

### 2. Context 上下文
*   **核心作用**：超时控制、取消信号传播、Request Scoped 数据传递。
*   **源码分析**：`backend/main.go` 的 SSE 推送接口：
    ```go
    ctx := c.Request.Context()
    select {
    case <-ctx.Done(): // 监听客户端断开
        return
    }
    ```
    如果不监听 `ctx.Done()`，客户端断开后，Goroutine 仍会空跑，导致**Goroutine 泄漏**。

### 3. 并发模式实战
*   **Semaphore (信号量)**：控制并发数。
    *   *代码位置*：`backend/lan/scanner.go`
    *   ```go
        sem := make(chan struct{}, 50) // 限制最大 50 并发
        for _, ip := range ips {
            sem <- struct{}{}
            go func() {
                defer func() { <-sem }()
                // scan logic...
            }()
        }
        ```
    *   **面试题**：*如果不加限制会怎样？* 会瞬间创建 255+ 个 Goroutine，导致大量 ICMP 包瞬间发出，可能触发系统 `ulimit` 限制，或被防火墙判定为 DDoS 攻击从而丢包。

---

## 第三部分：项目实战与系统设计 (System Monitor 专项)

### 1. 项目难点：如何解决容器化部署的监控失真？
*   **问题**：Docker 容器通过 Namespace (PID, MNT, NET) 隔离。直接在容器内读取 `/proc/meminfo` 或 `/proc/net/dev` 拿到的是容器环境的数据，而非宿主机。
*   **解决方案 (Host Mapping)**：
    *   **原理**：利用 Linux `bind mount` 机制，将宿主机的 `/proc`, `/sys` 目录挂载到容器内 (如 `/host/proc`)。
    *   **代码实现**：
        *   环境变量 `HOST_PROC` 指向挂载路径。
        *   `gopsutil` 库支持通过 `os.Setenv("HOST_PROC", ...)` 重定向读取路径。
        *   磁盘监控需特殊处理：`/host/proc/mounts` 中的路径是宿主机的 (如 `/mnt/data`)，在容器内不可访问，需要映射回 `/hostfs/mnt/data`。

### 2. 项目难点：实时数据推送方案选型
*   **对比 WebSocket vs SSE**：
    *   **WebSocket**：全双工，适合 IM 聊天、即时游戏。协议复杂 (握手/心跳/Frame)，需专用库。
    *   **SSE (Server-Sent Events)**：单向 (Server->Client)，基于 HTTP。
*   **为什么选 SSE？**
    *   监控场景是典型的**单向高频推送**。
    *   SSE 自带**断线重连** (Browser Native)。
    *   Go 实现极其简单 (设置 Header `text/event-stream` + `Flush`)，无额外依赖。
*   **代码亮点**：`backend/main.go` 中使用了 `c.Writer.Flush()` 确保数据不被缓冲，实时送达。

### 3. 项目难点：网络安全审计实现
*   **技术链**：`net.Connections` -> `Process Lookup` -> `GeoIP`。
*   **关键点**：
    *   **反查进程**：通过 socket inode 或系统调用获取 PID，再读取 `/proc/{pid}/comm` 获取进程名。
    *   **性能优化**：IP 地理库 (GeoLite2) 是二进制文件，启动时一次性加载到内存 (`backend/metrics/collector.go: InitGeo`)，查询耗时微秒级，避免每次 IO。

---

## 第四部分：常用工具与性能调优

### 1. Pprof 性能分析
*   **类型**：
    *   **CPU Profile**：采样分析 CPU 耗时热点。
    *   **Heap Profile**：分析内存分配，定位内存泄漏。
    *   **Goroutine Profile**：查看 G 数量及阻塞位置，定位死锁或泄漏。
*   **使用**：`import _ "net/http/pprof"`，访问 `/debug/pprof`。

### 2. 竞态检测 (Race Detector)
*   **命令**：`go run -race main.go`。
*   **原理**：在编译时插入指令，运行时记录内存访问，检测**多线程对同一内存地址的非同步读写**。
*   **注意**：开启 race 会有 5-10 倍性能损耗，生产环境慎用。

---

## 第五部分：LeetCode 高频题 (Golang 版)

1.  **交替打印 FooBar**：使用 Channel 或 `sync.Cond` 实现两个 G 协同工作。
2.  **LRU 缓存**：`map` + `container/list` (双向链表)。
3.  **生产者消费者模型**：带缓冲 Channel。
4.  **反转链表**：双指针。
5.  **二叉树层序遍历**：Slice 作为队列实现 BFS。
