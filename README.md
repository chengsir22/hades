# Hades

论文：https://riak.com/assets/bitcask-intro.pdf

go实现基于bitcask模型，兼容redis数据结构和协议的高性能kv存储引擎

## 整体架构

![img](./assets/(null)-20240103193706385.(null))<img src="./assets/(null)-20240103193706429.(null)" alt="img" style="zoom:50%;" />

bitcask 是一种高性能的持久化存储引擎，其基本原理是采用了预写日志的数据存储方式，每一条数据在写入时首先会追加写入到数据文件中，然后更新内存索引，内存索引存储了 Key 和 Value 在磁盘上的位置，读取数据时，会先从内存中根据 kev 找到对应 Value 的位置，然后再从磁盘中取出实际的 Value。基于这种模型，其读写性能都非常高效快速，因为每次写入实际上都是一次顺序IO 操作，然后更新内存。每次读取也是直接从内存中找到对应数据在磁盘上的位置。

**内存****设计** 内存索引 可以选择BTree、跳表、红黑树等（这里实现跳表，简单点😊），统一提供抽象接口Indexer，支持Put、Get、Delete

**磁盘设计** 定义IOSelector接口，将IO操作的接口进行抽象、方便接入不同IO类型，用于fileio和mmap的io选择器，对标准文件操作api例如read、write、sync、close、delete进行封装

## 内存索引-skiplist

跳表的实现方式很多，这里实现一个简单的，实现可以参考 [Golang 实现 Redis(5): 使用跳表实现 SortedSet - -Finley- - 博客园](https://www.cnblogs.com/Finley/p/12854599.html) [注意不同redis 跳表 key是score int，kv里的key就是string值]

![img](./assets/(null)-20240103193706354.(null))

```Go
type Node struct {
    key     []byte
    value   interface{}
    forward []*Node // 各层的下一个指针
}

type SkipList struct {
    head   *Node
    level  int16
    length int
    lock   *sync.RWMutex
}
```

![img](./assets/(null)-20240103193706451.(null))

```C++
// Put 向索引中存储 key 对应的数据位置信息, 如果键已存在，更新值并返回旧值
func (sl *SkipList) Put(key []byte, pos *data.LogRecordPos) *data.LogRecordPos {
    sl.lock.Lock()
    defer sl.lock.Unlock()

    update := make([]*Node, maxLevel) // 记录每层需要更新的节点
    current := sl.head

    // 从最高层开始查找 【关键代码】
    for i := sl.level - 1; i >= 0; i-- {
        // 在当前层查找插入位置
        for current.forward[i] != nil && bytes.Compare(current.forward[i].key, key) < 0 {
            current = current.forward[i] // current.forward[i].key < key
        }
        update[i] = current
    }

    if current.forward[0] != nil && bytes.Equal(current.forward[0].key, key) {
        // 如果键已存在，更新值并返回旧值
        oldVal := current.forward[0].value
        current.forward[0].value = pos
        return oldVal.(*data.LogRecordPos)
    }

    level := randomLevel()
    if level > sl.level {
        // 如果新节点的层数大于当前层数，需要更新 update 切片
        for i := sl.level; i < level; i++ {
            update[i] = sl.head
        }
        sl.level = level
    }

    newNode := newNode(key, pos, level)
    for i := int16(0); i < level; i++ {
        // 更新节点的各层指针
        newNode.forward[i] = update[i].forward[i]
        update[i].forward[i] = newNode
    }
    sl.length++
    return nil
}
```

节点level纯随机有弊端，思考 可以将热点数据增加level，冷数据降低level

## 读写数据

```Go
type LogRecord struct {
    Key   []byte
    Value []byte
    Type  LogRecordType
}
```

logrecord序列化为encRecord，追加数据到活跃文件，如果活跃文件为空，则创建活跃文件（数据库初始化时），如果活跃文件WriteOff+len(encRecord)>=文件阀值DataFileSize，则持久化当前活跃文件为旧文件，创建新的活跃文件；将encRecord写入活跃文件。

删除流程 删除只是将内存索引删除了，磁盘中新增一条带墓碑值的LogRecord，并未实际删除磁盘数据（merge时才真正删除）

## **数据库启动** 

启动流程主要的步骤有两个，一是加载数据目录中的文件，打开其文件描述符，二是遍历数据文件中的内容，构建内存索引。

### **读取LogRecord**

Header 部分的数据中，crc 占4字节，type 占一个字节，key size和value size 是变长的，从数据文件中读取的时候，我们会取最大的 header字节数，反序列化的时候，如果解码 key size和 value size 之后还有多余的字节，会自动忽略。

拿到header 之后，如果判断到 key size和 value size均为 0，则说明读取到了文件的末尾，我们直接返回一个EOF 的错误。否则，再根据 header 中的 key 和 value 的长度信息，判断其值是否大于 0，如果是的话，则说明存在 key 或者value.

将读偏移 offset加上 keySize 和valueSize 的总和，读出一个字节数组，这就是实际的 KeyValue 数据，填充到 LogRecord 结构体中。

最后，需要根据读出的信息，获取到其对应的校验值 CRC，判断和 header 中的 CRC 是否相等，只有完全相等才说明这是一条完整有效的数据，否则说明数据可能被破坏了。

![img](./assets/(null)-20240103193706421.(null))

```C++
// EncodeLogRecord 对 LogRecord 进行编码，返回字节数组及长度
//  +-------------+-------------+-------------+--------------+-------------+--------------+
//  | crc 校验值  |  type 类型   |    key size |   value size |      key    |      value   |
//  +-------------+-------------+-------------+--------------+-------------+--------------+
//      4字节          1字节        变长（最大5）   变长（最大5）     变长           变长
func EncodeLogRecord(logRecord *LogRecord) ([]byte, int64) {
    // 初始化一个 header 部分的字节数组
    header := make([]byte, maxLogRecordHeaderSize)

    // 第五个字节存储 Type
    header[4] = logRecord.Type
    var index = 5
    // 5 字节之后，存储的是 key 和 value 的长度信息
    // 使用变长类型，节省空间
    index += binary.PutVarint(header[index:], int64(len(logRecord.Key)))
    index += binary.PutVarint(header[index:], int64(len(logRecord.Value)))
    var size = index + len(logRecord.Key) + len(logRecord.Value)
    encBytes := make([]byte, size)

    // 将 header 部分的内容拷贝过来
    copy(encBytes[:index], header[:index])
    // 将 key 和 value 数据拷贝到字节数组中
    copy(encBytes[index:], logRecord.Key)
    copy(encBytes[index+len(logRecord.Key):], logRecord.Value)

    // 对整个 LogRecord 的数据进行 crc 校验
    crc := crc32.ChecksumIEEE(encBytes[4:])
    binary.LittleEndian.PutUint32(encBytes[:4], crc)

    return encBytes, int64(size)
}
```

## Merge

将活跃文件设置为旧文件，创建一个新的活跃文件，启动一个临时数据库实例，这个实例和正在运行的数据库实例互不冲突，因为它们是不同的进程。遍历旧文件，ReadLogRecord读取logrecord，和内存中的索引位置进行比较，如果有效则重写入merge文件和hint文件。（merge完成，增加一个标识merge完成的文件，否者无效的merge，删除merge目录）

如果是有效的 merge，将merge目录中的数据文件，还有一个对应的 hint 索引文件。将这些数据文件拷贝到原始数据目录中，然后把对应的 hint 索引文件也拷贝过去，并且把临时的merge 目录删除掉，这样下次启动的时候，便能够和原来的正常启动保持一致了。

Merge 优化：我们可以统计失效的数据量，只有当失效的数据占比达到某个比例，才进行merge操作

## WriteBatch 原子写

- 读未提交RU是指，一个事务还没提交时，它做的变更就能被别的事务看到。（脏读）
- 读提交RC是指，一个事务提交之后，它做的变更才会被其他事务看到。（不可重复读）
- 可重复读RR是指，一个事务执行过程中看到的数据，总是跟这个事务在启动时看到的数据是一致的。当然在可重复读隔离级别下，未提交变更对其他事务也是不可见的。（MySQL默认RR）（幻读）
- 串行化，顾名思义是对于同一行记录，“写”会加“写锁”，“读”会加“读锁”。当出现读写锁冲突的时候，后访问的事务必须等前一个事务执行完成，才能继续执行。

将用户的批量操作保存起来，保存到一个内存数据结构中（map），提供一个commit方法，将批量操作全部写入到磁盘中，并更新内存索引

这一批数据添加一个唯一标识，一般把它叫做序列号 Seq Number，即事务 ID。这个seq number 是全局递增的，每一个批次的数据在提交的时候都将获取一个 seq number，并且保证后面的 seq number 一定比前面获取的更大。

提交事务的时候，每一条日志记录LogRecord 都有一个 seq number，并且写到数据文件中，然后我们可以在这一批次的最后增加一个标识事务完成的日志记录。

在数据库启动的时候，如果判断到日志记录有序列号 seq number，那么我们先不直接更新内存索引，而是将它暂存起来，直到读到了一条标识事务完成的记录，说明事务是正常提交的，就可以将这一批数据都更新到内存索引中。

```Go
// WriteBatch 原子批量写数据，保证原子性
type WriteBatch struct {
   options       *settings.WriteBatchConfig  // 一个批次中最大数据量 和 每次事务提交是否持久化
   mu            *sync.Mutex
   db            *DB
   pendingWrites map[string]*data.LogRecord // 暂存用户写入的数据
}
```

Commit 方法是核心逻辑，我们需要拿到当前最新的 seq number，然后将其添加到LogRecord 中，这里采取的办法是将 seq number和 key 编码到一起，并且采用变长数组，尽量节省空间。

启动数据库的时候，不能直接拿到 LogRecord 就去更新内存索引了，因为LogRecord 有可能是无效的事务的数据，所以我们将其暂存起来，如果读到了一个标识事务完成的数据，才将暂存的对应的事务id 的数据更新到内存索引。

在遍历数据的时候，还需要更新对应的 seqNo，并找到最大的那个值，方便数据库启动之后，新的 WriteBatch 能够拿到最新的事务序列号。

## 文件优化

使用文件锁，保证bitcask只能单进程运行 https://github.com/gofrs/flock

持久化策略，可以在打开数据库的时候，增加一个配置项 BytesPerSync，每次写数据的时候，都记录一下累计写了多少个字节，如果累计值达到了 BytesPerSync，则进行持久化。

### Mmap

https://pkg.go.dev/golang.org/x/exp/mmap

mmap减少内核态到用户态的数据拷贝，使用mmap加速bitcask启动。打开数据文件的时候，我们将按照 MMap IO 的方式初始化IOManager，加载索引时读数据都会使用 mmap。加载索引完成后，我们需要重置我们的ioselector，因为 MMap 只是用于数据库启动，启动完成之后，要将lOManager 切换到原来的 IO 类型。

```C++
readerAt, err := mmap.Open(fileName)
readerAt.ReadAt(b, offset)
```

## 数据备份

只需要将数据目录拷贝到其他的位置，这样就算原有的目录损坏了，拷贝的目录中仍然存有备份的数据，可以直接在这个新的目录中启动 bitcask，保证数据不丢失。

```C++
// CopyDir 拷贝数据目录
func CopyDir(src, dest string, exclude []string) error {
    // 目标目标不存在则创建
    fmt.Println(src, dest)
    if _, err := os.Stat(dest); os.IsNotExist(err) {
       if err := os.MkdirAll(dest, os.ModePerm); err != nil {
          return err
       }
    }

    return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
       //fileName := strings.Replace(path, src, "", 1)
       fileName := filepath.Base(path)
       if fileName == "" {
          return nil
       }

       for _, e := range exclude {
          matched, err := filepath.Match(e, info.Name())
          if err != nil {
             return err
          }
          if matched {
             return nil
          }
       }

       if info.IsDir() {
          return os.MkdirAll(filepath.Join(dest, fileName), info.Mode())
       }

       data, err := os.ReadFile(filepath.Join(src, fileName))
       if err != nil {
          return err
       }
       return os.WriteFile(filepath.Join(dest, fileName), data, info.Mode())
    })
}
```

# Redis

兼容5种redis协议 string、hash、set、list、Sorted set，很简单 直接看代码中实现。

```C++
// 元数据
type metadata struct {
   dataType byte   // 数据类型
   expire   int64  // 过期时间
   version  int64  // 版本号
   size     uint32 // 数据量
   
   head     uint64 // List 数据结构专用 =U64_MAX/2
   tail     uint64 // List 数据结构专用
}
```

String key---type|expire|payload

Hash key---元数据 key|version|filed------value

首先根据 key 查询元数据，如果不存在的话则说明 key 不存在，否则判断 value 的类型，如果 value 的类型不是Hash，则直接返回对应的错误然后我们根据 key 和元数据中的 version 字段，以及 field 编码出一个 key，然后根据这个 key 去获取实际的value

Set key---元数据 key|version|member------NULL

List key---元数据 key|version|index------value 

Zset key---元数据 key|version|member------score key|version|score|member-----NULL 

version用于快速删除，version 是递增的，假如 key 被删除之后，我们又重新添加了这个 key，这时候会分配一个新的 version，和之前的version 不一样，所以我们就查不到之前的旧的数据。

## resp协议解析

可以自己实现参考Godis，这里直接使用https://github.com/tidwall/redcon

# refrence

MySQL 是怎样运行的：从根儿上理解 MySQL 

**RoseDB Godis** **bitcask-kv**
