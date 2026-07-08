# API 文档

## 接口列表

### GET /users

获取用户列表。

```go
func GetUsers() ([]User, error) {
    return db.Query("SELECT * FROM users")
}
```

### POST /users

创建新用户。

```bash
curl -X POST /users -d '{"name": "test"}'
```

## 错误码

| 码 | 含义 |
|----|------|
| 200 | 成功 |
| 400 | 参数错误 |
| 500 | 服务器错误 |
