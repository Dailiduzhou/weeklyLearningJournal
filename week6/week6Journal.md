# Week6
## `mysql`
- `Docker`安装：官方有mysql镜像，只要保证镜像源正确配置即可下载。
- `Windows`安装：zip Archive版，记得在**环境变量**配置`mysql`，还有千万别忘了`root`的密码。
  
### `transaction`
类似`Redis`的`Pipeline`，都是将一系列操作统一执行，减少与数据库交流的性能开销的方法。
符合`ACID`原则。

手动开启`transaction`还是太原始了。
一般可以结合`gorm`包，
调用`DB.transaction`函数。
示例：
``` go
config.DB.Transaction(func(tx *gorm.DB) error {
    // 在事务中执行操作
    if err := tx.Create(&user1).Error; err != nil {
        return err // 回滚事务
    }
    if err := tx.Create(&user2).Error; err != nil {
        return err // 回滚事务
    }
    return nil // 提交事务
})
```
对于`func(tx *gorm.DB) error`返回`error`的情况，自动回滚事务。
不需要手动管理，非常便利。