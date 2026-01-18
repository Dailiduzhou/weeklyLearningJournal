# SeaORM 数据库操作详解

本文档详细说明 `mvc_seaorm` 项目中数据库部分的代码逻辑。

## 一、核心概念

### 1. SeaORM 实体模型 (src/entity.rs)

```rust
#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Serialize, Deserialize)]
#[sea_orm(table_name = "users")]  // 映射到 users 表
pub struct Model {
    #[sea_orm(primary_key)]      // 主键标识
    pub id: i64,
    #[sea_orm(unique)]           // 唯一约束
    pub username: String,
    pub password: String,
    pub name: String,
    pub created_at: Option<chrono::NaiveDateTime>,
    pub updated_at: Option<chrono::NaiveDateTime>,
}
```

**关键点**：
- `DeriveEntityModel` 宏自动生成 `Entity`、`Column`、`PrimaryKey` 等类型
- `Model` 表示数据库记录
- `ActiveModel` 用于插入/更新操作（通过 `Set()` 标记修改字段）
- 编译时生成类型安全的查询 API

### 2. 数据库连接 (src/main.rs)

```rust
let db = Database::connect("mysql://root:123456@localhost:3307/ginserver")
    .await
    .expect("数据库连接失败");
```

**与 sqlx 对比**：
```rust
// sqlx
let pool = MySqlPoolOptions::new().connect(db_url).await?;

// SeaORM
let db = Database::connect(db_url).await?;
```

## 二、CRUD 操作详解

### 1. 创建 (Create) - 注册用户

```rust
#[post("/api/users")]
async fn register(form: web::Form<RegisterForm>, state: web::Data<AppState>) -> impl Responder {
    // 1. 密码加密
    let hashed = hash(&form.password, DEFAULT_COST)?;
    
    // 2. 构建 ActiveModel（准备插入的数据）
    let new_user = UserActiveModel {
        username: Set(form.username.clone()),  // Set() 标记为"要插入"
        password: Set(hashed),
        name: Set(form.name.clone()),
        created_at: Set(Some(chrono::Local::now().naive_local())),
        updated_at: Set(Some(chrono::Local::now().naive_local())),
        ..Default::default()  // id 自增，不需要手动设置
    };
    
    // 3. 执行插入
    new_user.insert(&state.db).await?;
}
```

**等价 SQL**：
```sql
INSERT INTO users (username, password, name, created_at, updated_at)
VALUES (?, ?, ?, NOW(), NOW());
```

**sqlx 对比**：
```rust
// sqlx 需要手写 SQL
sqlx::query("INSERT INTO users (username, password, name, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())")
    .bind(&username)
    .bind(&hashed)
    .bind(&name)
    .execute(&db).await?;
```

### 2. 查询 (Read) - 登录验证

```rust
#[post("/api/users/login")]
async fn login(form: web::Form<LoginForm>, state: web::Data<AppState>) -> impl Responder {
    // 1. 构建查询（链式 API）
    let user = UserEntity::find()           // SELECT * FROM users
        .filter(UserColumn::Username.eq(&form.username))  // WHERE username = ?
        .one(&state.db)                     // LIMIT 1
        .await?;
    
    // 2. 模式匹配结果
    let user = match user {
        Ok(Some(u)) => u,      // 找到用户
        Ok(None) => return HttpResponse::NotFound(),  // 未找到
        Err(_) => return HttpResponse::InternalServerError(),
    };
    
    // 3. 验证密码
    if !verify(&form.password, &user.password).unwrap_or(false) {
        return HttpResponse::Forbidden();
    }
}
```

**等价 SQL**：
```sql
SELECT id, username, password, name, created_at, updated_at
FROM users
WHERE username = ?
LIMIT 1;
```

**查询方法对比**：

| SeaORM                            | SQL                    | 说明                  |
| --------------------------------- | ---------------------- | --------------------- |
| `.one()`                          | `LIMIT 1`              | 返回 `Option<Model>`  |
| `.all()`                          | 无 LIMIT               | 返回 `Vec<Model>`     |
| `.filter(Column::Eq(...))`        | `WHERE ... = ?`        | 相等条件              |
| `.filter(Column::Contains(...))` | `WHERE ... LIKE %?%`   | 模糊匹配              |

### 3. 更新 (Update) - 修改资料

```rust
#[put("/api/users/profiles")]
async fn change_profile(form: web::Form<ChangeProfileForm>, state: web::Data<AppState>) -> impl Responder {
    // 1. 查询现有用户
    let user = UserEntity::find()
        .filter(UserColumn::Username.eq(&session_username))
        .one(&state.db).await?;
    
    // 2. 转换为 ActiveModel（可修改状态）
    let mut active_user: UserActiveModel = user.into();
    
    // 3. 选择性更新字段
    if let Some(nu) = &form.newusername { 
        active_user.username = Set(nu.clone());  // 只更新 username
    }
    if let Some(nn) = &form.newname { 
        active_user.name = Set(nn.clone());      // 只更新 name
    }
    active_user.updated_at = Set(Some(chrono::Local::now().naive_local()));
    
    // 4. 执行更新
    active_user.update(&state.db).await?;
}
```

**等价 SQL（两个字段都更新时）**：
```sql
UPDATE users
SET username = ?, name = ?, updated_at = NOW()
WHERE id = ?;
```

**关键优势**：
- SeaORM 自动生成 `SET` 子句，只更新被 `Set()` 标记的字段
- sqlx 需要手动拼接 SQL，容易出错

**sqlx 对比（需手动处理可选更新）**：
```rust
// sqlx 需要手动构建动态 SQL
let query = if form.newusername.is_some() && form.newname.is_some() {
    sqlx::query("UPDATE users SET username = ?, name = ?, updated_at = NOW() WHERE username = ?")
        .bind(form.newusername.as_ref().unwrap())
        .bind(form.newname.as_ref().unwrap())
        .bind(&username)
} else if form.newusername.is_some() {
    sqlx::query("UPDATE users SET username = ?, updated_at = NOW() WHERE username = ?")
        .bind(form.newusername.as_ref().unwrap())
        .bind(&username)
} else {
    sqlx::query("UPDATE users SET name = ?, updated_at = NOW() WHERE username = ?")
        .bind(form.newname.as_ref().unwrap())
        .bind(&username)
};
query.execute(&db).await?;
```

### 4. 查询所有 - 管理员列表

```rust
#[get("/api/admin/users")]
async fn users_data(state: web::Data<AppState>) -> impl Responder {
    let users = UserEntity::find()  // SELECT * FROM users
        .all(&state.db)             // 获取所有记录
        .await?;
    
    HttpResponse::Ok().json(serde_json::json!({
        "users": users,
        "count": users.len()
    }))
}
```

**等价 SQL**：
```sql
SELECT id, username, password, name, created_at, updated_at
FROM users;
```

## 三、ActiveModel 状态机

SeaORM 使用状态标记来跟踪字段修改：

```rust
pub enum ActiveValue<T> {
    Unchanged(T),  // 未修改
    Set(T),        // 新值（需要更新/插入）
    NotSet,        // 未设置（插入时忽略，更新时不改）
}
```

**示例**：
```rust
let mut user: UserActiveModel = existing_user.into();
user.username = Set("newname".to_string());  // 标记为修改
user.name = NotSet;                          // 不修改此字段
user.update(&db).await?;  // 只更新 username 和 updated_at
```

## 四、类型安全查询

### 编译期保证
```rust
// ✅ 编译通过
UserEntity::find()
    .filter(UserColumn::Username.eq("test"))  // Username 是合法列
    .one(&db).await?;

// ❌ 编译错误
UserEntity::find()
    .filter(UserColumn::NonExistent.eq("test"))  // 不存在的列
    .one(&db).await?;
```

### 查询构建器
```rust
UserEntity::find()
    .filter(UserColumn::Name.contains("张"))       // LIKE '%张%'
    .filter(UserColumn::Id.gt(10))                // id > 10
    .order_by_desc(UserColumn::CreatedAt)         // ORDER BY created_at DESC
    .limit(10)                                    // LIMIT 10
    .offset(20)                                   // OFFSET 20
    .all(&db).await?;
```

## 五、错误处理

```rust
match UserEntity::find().one(&db).await {
    Ok(Some(user)) => { /* 找到用户 */ },
    Ok(None) => { /* 未找到 */ },
    Err(DbErr::RecordNotFound(_)) => { /* 同上 */ },
    Err(DbErr::Query(e)) => { /* SQL 错误 */ },
    Err(DbErr::Conn(e)) => { /* 连接错误 */ },
    Err(_) => { /* 其他错误 */ },
}
```

## 六、性能对比

| 操作         | sqlx   | SeaORM                |
| ------------ | ------ | --------------------- |
| 简单查询     | 100%   | ~95% (轻微开销)       |
| 批量插入     | 100%   | ~90% (ORM 转换)       |
| 复杂关联查询 | 100%   | ~85% (需预加载优化)   |
| 开发效率     | 中     | 高                    |
| 类型安全     | 宏验证 | 完全类型安全          |

## 七、最佳实践

### 1. 使用事务
```rust
let txn = db.begin().await?;
UserEntity::insert(user1).exec(&txn).await?;
UserEntity::insert(user2).exec(&txn).await?;
txn.commit().await?;
```

### 2. 预加载关联（如果有外键）
```rust
// 假设 User has_many Post
let users_with_posts = UserEntity::find()
    .find_with_related(PostEntity)  // JOIN posts
    .all(&db).await?;
```

### 3. 原生 SQL 回退
```rust
// 当 ORM 不够用时，SeaORM 支持原生查询
let users: Vec<User> = User::find_by_statement(
    Statement::from_sql_and_values(
        DbBackend::MySql,
        "SELECT * FROM users WHERE YEAR(created_at) = ?",
        vec![2025.into()]
    )
)
.all(&db).await?;
```

## 八、迁移建议

从 sqlx 迁移到 SeaORM：

1. **定义实体**：根据现有表结构创建 `Model`
2. **替换查询**：将 `sqlx::query!` 改为 `Entity::find()`
3. **替换插入**：将 `sqlx::query!` 改为 `ActiveModel::insert()`
4. **替换更新**：先查询，转 `ActiveModel`，修改，`update()`
5. **测试覆盖**：确保所有数据库操作有单元测试

## 九、总结

**使用 SeaORM 的场景**：
- ✅ 复杂业务逻辑，需要大量 CRUD
- ✅ 团队习惯 ORM 而非原生 SQL
- ✅ 需要编译期类型检查
- ✅ 关联查询多

**使用 sqlx 的场景**：
- ✅ 性能敏感，需要极致优化
- ✅ 复杂 SQL（窗口函数、CTE 等）
- ✅ 团队 SQL 技能强
- ✅ 简单 CRUD，不需要 ORM 抽象

本项目展示了 SeaORM 的典型用法，适合作为学习 Rust ORM 的起点。
