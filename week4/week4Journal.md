# Week4
## `swagger`
在`VS Code`装了`OpenAPI(Swagger) Editor`, `Swagger Viewer`的扩展。

写完服务端就扔给`deepseek`生成注释了，再根据注释通过`go-swagger`生成了`yaml`,`json`然后报了好多错。
再丢回去，改完果然好了。看来AI的水平不在我之下（bushi）。
看来`Qwen`对生成代码和swagger文档的能力强一点。
在`Qwen`帮助下，把路由改为了`RESTful`风格。

还没能实现控制swagger文档访问权的功能。
## `gin`框架
`gin`框架整合了原生的`net/http`包，对路由的操作变得简单、方便。显示`json`、`HTML`，提取表单元素也变得轻松。