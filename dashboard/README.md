# Permission Center Dashboard

权限中心的 React + TypeScript + TDesign 管理界面，包含 NexusAuth 登录、角色管理、菜单/按钮资源树与角色授权操作。

```sh
yarn install
yarn dev
```

开发服务器监听 `http://localhost:5274`，并将 `/api` 代理到 `http://localhost:8080`。NexusAuth 客户端的开发回调地址应登记为 `http://localhost:8080/signin-oidc`；Go 后端完成认证后会跳转到 `http://localhost:5274/auth/callback`。
