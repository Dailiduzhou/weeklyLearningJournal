use actix_session::{storage::CookieSessionStore, Session, SessionMiddleware};
use actix_web::cookie::Key;
use actix_web::middleware::Logger;
use actix_web::{get, post, put, web, App, HttpResponse, HttpServer, Responder};
use bcrypt::{hash, verify, DEFAULT_COST};
use chrono::NaiveDateTime;
use serde::{Deserialize, Serialize};
use sqlx::{mysql::MySqlPoolOptions, MySql, Pool};
use tera::{Context, Tera};

#[derive(sqlx::FromRow, Serialize)]
struct User {
    id: i64,
    username: String,
    password: String,
    name: String,
    created_at: Option<NaiveDateTime>,
    updated_at: Option<NaiveDateTime>,
}

#[derive(Clone)]
struct AppState {
    db: Pool<MySql>,
    tera: Tera,
}

#[get("/api/users")]
async fn page_register(tmpl: web::Data<Tera>) -> impl Responder {
    let mut ctx = Context::new();
    ctx.insert("title", "用户注册");
    match tmpl.render("register.html", &ctx) {
        Ok(html) => HttpResponse::Ok().content_type("text/html").body(html),
        Err(e) => HttpResponse::InternalServerError().json(serde_json::json!({
            "error": e.to_string()
        })),
    }
}

#[derive(Deserialize)]
struct RegisterForm {
    username: String,
    password: String,
    name: String,
}

#[post("/api/users")]
async fn register(form: web::Form<RegisterForm>, state: web::Data<AppState>) -> impl Responder {
    if form.username.is_empty() || form.password.is_empty() || form.name.is_empty() {
        return HttpResponse::BadRequest()
            .json(serde_json::json!({"error": "用户名、密码和姓名不能为空"}));
    }

    let hashed = match hash(&form.password, DEFAULT_COST) {
        Ok(h) => h,
        Err(_) => {
            return HttpResponse::InternalServerError()
                .json(serde_json::json!({"error": "密码加密失败"}))
        }
    };

    let res = sqlx::query(
        "INSERT INTO users (username, password, name, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())"
    )
    .bind(&form.username)
    .bind(&hashed)
    .bind(&form.name)
    .execute(&state.db).await;
    match res {
        Ok(_) => HttpResponse::Created()
            .json(serde_json::json!({"username": form.username, "name": form.name})),
        Err(_) => {
            HttpResponse::InternalServerError().json(serde_json::json!({"error": "创建用户失败"}))
        }
    }
}

#[get("/api/users/login")]
async fn page_login(tmpl: web::Data<Tera>) -> impl Responder {
    let mut ctx = Context::new();
    ctx.insert("title", "用户登录");
    match tmpl.render("login.html", &ctx) {
        Ok(html) => HttpResponse::Ok().content_type("text/html").body(html),
        Err(e) => {
            HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()}))
        }
    }
}

#[derive(Deserialize)]
struct LoginForm {
    username: String,
    password: String,
}

#[post("/api/users/login")]
async fn login(
    form: web::Form<LoginForm>,
    session: Session,
    state: web::Data<AppState>,
) -> impl Responder {
    if form.username.is_empty() || form.password.is_empty() {
        return HttpResponse::BadRequest()
            .json(serde_json::json!({"error": "用户名和密码不能为空"}));
    }

    let rec = sqlx::query_as::<_, User>(
        "SELECT id, username, password, name, created_at, updated_at FROM users WHERE username = ?",
    )
    .bind(&form.username)
    .fetch_optional(&state.db)
    .await;

    let user = match rec {
        Ok(Some(u)) => u,
        Ok(None) => {
            return HttpResponse::NotFound().json(serde_json::json!({"error": "用户不存在"}))
        }
        Err(_) => return HttpResponse::InternalServerError().finish(),
    };

    if !verify(&form.password, &user.password).unwrap_or(false) {
        return HttpResponse::Forbidden().json(serde_json::json!({"error": "密码错误"}));
    }

    if session.insert("username", &user.username).is_err() {
        return HttpResponse::InternalServerError()
            .json(serde_json::json!({"error": "会话创建失败"}));
    }

    HttpResponse::Ok().json(serde_json::json!({"username": user.username, "name": user.name}))
}

#[get("/api/users/me")]
async fn page_profiles(
    session: Session,
    tmpl: web::Data<Tera>,
    state: web::Data<AppState>,
) -> impl Responder {
    let username: Option<String> = session.get("username").unwrap_or(None);
    if username.is_none() {
        return HttpResponse::Found()
            .append_header(("Location", "/api/users/login"))
            .finish();
    }
    let user = sqlx::query_as::<_, User>(
        "SELECT id, username, password, name, created_at, updated_at FROM users WHERE username = ?",
    )
    .bind(username.clone().unwrap())
    .fetch_one(&state.db)
    .await;

    let user = match user {
        Ok(u) => u,
        Err(_) => {
            let _ = session.purge();
            return HttpResponse::Found()
                .append_header(("Location", "/api/users/login"))
                .finish();
        }
    };

    let mut ctx = Context::new();
    ctx.insert("username", &user.username);
    ctx.insert("name", &user.name);
    match tmpl.render("profiles.html", &ctx) {
        Ok(html) => HttpResponse::Ok().content_type("text/html").body(html),
        Err(e) => {
            HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()}))
        }
    }
}

#[post("/api/users/logout")]
async fn logout(session: Session) -> impl Responder {
    let _ = session.purge();
    HttpResponse::Ok().json(serde_json::json!({"message": "登出成功"}))
}

#[get("/api/users/profiles")]
async fn page_change_profile(session: Session, tmpl: web::Data<Tera>) -> impl Responder {
    let username: Option<String> = session.get("username").unwrap_or(None);
    if username.is_none() {
        return HttpResponse::Found()
            .append_header(("Location", "/api/users/login"))
            .finish();
    }

    let mut ctx = Context::new();
    ctx.insert("username", &username.unwrap());
    match tmpl.render("changeprofiles.html", &ctx) {
        Ok(html) => HttpResponse::Ok().content_type("text/html").body(html),
        Err(e) => {
            HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()}))
        }
    }
}

#[derive(Deserialize)]
struct ChangeProfileForm {
    newusername: Option<String>,
    newname: Option<String>,
}

#[put("/api/users/profiles")]
async fn change_profile(
    session: Session,
    form: web::Form<ChangeProfileForm>,
    state: web::Data<AppState>,
) -> impl Responder {
    let session_username: Option<String> = session.get("username").unwrap_or(None);
    if session_username.is_none() {
        return HttpResponse::Unauthorized().json(serde_json::json!({"error": "未认证,请先登录"}));
    }
    let user = sqlx::query_as::<_, User>(
        "SELECT id, username, password, name, created_at, updated_at FROM users WHERE username = ?",
    )
    .bind(session_username.clone().unwrap())
    .fetch_one(&state.db)
    .await;
    let user = match user {
        Ok(u) => u,
        Err(_) => {
            return HttpResponse::Forbidden().json(serde_json::json!({"error": "用户不存在"}))
        }
    };

    if form.newusername.is_none() && form.newname.is_none() {
        return HttpResponse::BadRequest().json(serde_json::json!({"error": "未检测到有效更改"}));
    }
    if let Some(newu) = &form.newusername {
        if *newu == user.username {
            return HttpResponse::BadRequest()
                .json(serde_json::json!({"error": "新用户名与原用户名相同"}));
        }
    }
    if let Some(newn) = &form.newname {
        if *newn == user.name {
            return HttpResponse::BadRequest()
                .json(serde_json::json!({"error": "新姓名与原姓名相同"}));
        }
    }

    let res = if form.newusername.is_some() && form.newname.is_some() {
        sqlx::query(
            "UPDATE users SET username = ?, name = ?, updated_at = NOW() WHERE username = ?",
        )
        .bind(form.newusername.as_ref().unwrap())
        .bind(form.newname.as_ref().unwrap())
        .bind(&user.username)
        .execute(&state.db)
        .await
    } else if form.newusername.is_some() {
        sqlx::query("UPDATE users SET username = ?, updated_at = NOW() WHERE username = ?")
            .bind(form.newusername.as_ref().unwrap())
            .bind(&user.username)
            .execute(&state.db)
            .await
    } else {
        sqlx::query("UPDATE users SET name = ?, updated_at = NOW() WHERE username = ?")
            .bind(form.newname.as_ref().unwrap())
            .bind(&user.username)
            .execute(&state.db)
            .await
    };
    match res {
        Ok(_) => {
            if let Some(newu) = &form.newusername {
                let _ = session.insert("username", newu);
            }
            HttpResponse::Ok().json(serde_json::json!({"message": "用户信息更新成功"}))
        }
        Err(_) => HttpResponse::InternalServerError()
            .json(serde_json::json!({"error": "更新用户信息失败"})),
    }
}

#[get("/api/users/password")]
async fn page_change_password(session: Session, tmpl: web::Data<Tera>) -> impl Responder {
    let username: Option<String> = session.get("username").unwrap_or(None);
    if username.is_none() {
        return HttpResponse::Found()
            .append_header(("Location", "/login"))
            .finish();
    }
    let mut ctx = Context::new();
    ctx.insert("title", "修改密码");
    match tmpl.render("changepassword.html", &ctx) {
        Ok(html) => HttpResponse::Ok().content_type("text/html").body(html),
        Err(e) => {
            HttpResponse::InternalServerError().json(serde_json::json!({"error": e.to_string()}))
        }
    }
}

#[derive(Deserialize)]
struct ChangePasswordForm {
    oldpassword: String,
    password: String,
    password1: String,
}

#[put("/api/users/password")]
async fn change_password(
    session: Session,
    form: web::Form<ChangePasswordForm>,
    state: web::Data<AppState>,
) -> impl Responder {
    let username: Option<String> = session.get("username").unwrap_or(None);
    if username.is_none() {
        return HttpResponse::Unauthorized().json(serde_json::json!({"error": "未认证,请先登录"}));
    }

    if form.oldpassword.is_empty() || form.password.is_empty() || form.password1.is_empty() {
        return HttpResponse::BadRequest()
            .json(serde_json::json!({"error": "所有密码字段不能为空"}));
    }

    let user = sqlx::query_as::<_, User>(
        "SELECT id, username, password, name, created_at, updated_at FROM users WHERE username = ?",
    )
    .bind(username.clone().unwrap())
    .fetch_one(&state.db)
    .await;

    let user = match user {
        Ok(u) => u,
        Err(_) => return HttpResponse::NotFound().json(serde_json::json!({"error": "用户不存在"})),
    };

    if !verify(&form.oldpassword, &user.password).unwrap_or(false) {
        return HttpResponse::Forbidden().json(serde_json::json!({"error": "原密码错误"}));
    }

    if form.password != form.password1 {
        return HttpResponse::BadRequest()
            .json(serde_json::json!({"error": "两次输入的新密码不一致"}));
    }

    let hashed = match hash(&form.password, DEFAULT_COST) {
        Ok(h) => h,
        Err(_) => {
            return HttpResponse::InternalServerError()
                .json(serde_json::json!({"error": "密码加密失败"}))
        }
    };

    let res = sqlx::query("UPDATE users SET password = ?, updated_at = NOW() WHERE username = ?")
        .bind(&hashed)
        .bind(username.unwrap())
        .execute(&state.db)
        .await;
    match res {
        Ok(_) => HttpResponse::Ok().json(serde_json::json!({"message": "密码修改成功"})),
        Err(_) => HttpResponse::InternalServerError()
            .json(serde_json::json!({"error": "更新用户信息失败"})),
    }
}

#[get("/api/admin/users")]
async fn users_data(session: Session, state: web::Data<AppState>) -> impl Responder {
    let username: Option<String> = session.get("username").unwrap_or(None);
    if username.is_none() {
        return HttpResponse::Unauthorized().json(serde_json::json!({"error": "未认证,请先登录"}));
    }

    if username.as_deref() != Some("admin") {
        return HttpResponse::Forbidden()
            .json(serde_json::json!({"error": "权限不足,需要管理员权限"}));
    }
    let users = sqlx::query_as::<_, User>(
        "SELECT id, username, password, name, created_at, updated_at FROM users",
    )
    .fetch_all(&state.db)
    .await;
    match users {
        Ok(list) => {
            HttpResponse::Ok().json(serde_json::json!({"users": list, "count": list.len()}))
        }
        Err(_) => HttpResponse::InternalServerError()
            .json(serde_json::json!({"error": "获取用户信息失败"})),
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init();
    let key = Key::generate();

    // MySQL settings same as Go config
    let db_url = "mysql://root:123456@localhost:3307/ginserver";
    let pool = MySqlPoolOptions::new()
        .max_connections(5)
        .connect(db_url)
        .await
        .expect("数据库连接失败");

    // init templates (copy from week6/mvc/template)
    let mut tera = Tera::new("templates/**/*").expect("模板加载失败");

    let state = AppState {
        db: pool,
        tera: tera.clone(),
    };

    log::info!("服务器启动在 http://localhost:8080");
    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(state.clone()))
            .app_data(web::Data::new(tera.clone()))
            .wrap(Logger::default())
            .wrap(SessionMiddleware::new(
                CookieSessionStore::default(),
                key.clone(),
            ))
            .service(page_register)
            .service(register)
            .service(page_login)
            .service(login)
            .service(page_profiles)
            .service(logout)
            .service(page_change_profile)
            .service(change_profile)
            .service(page_change_password)
            .service(change_password)
            .service(users_data)
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}
