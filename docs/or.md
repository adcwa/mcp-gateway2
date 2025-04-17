mcp-gateway

参考最新 higress 功能，开发一个 mcp 网关
mcp 网关提供 MCP Server 统一托管能力，可以帮助 AI Agent 快速对接各类数据源。通过 MCP Server，AI Agent 可以方便地访问数据库、REST API 等外部服务，无需关心具体的连接细节。其中，数据库对接能力是网关内置的能力；而对于 REST API，任何外部 REST API 都可以通过简单的配置转换成 MCP Server 

配置 MCP Server 的方式可以参考
```
server:
  allowTools:
  - "get-user"
  name: "random-user-server"
tools:
- description: "Get random user information"
  name: "get-user"
  requestTemplate:
    method: "GET"
    url: "https://randomuser.me/api/"
  responseTemplate:
    body: |-
      # User Information
      {{- with (index .results 0) }}
      - **Name**: {{.name.first}} {{.name.last}}
      - **Email**: {{.email}}
      - **Location**: {{.location.city}}, {{.location.country}}
      - **Phone**: {{.phone}}
      {{- end }}
```
核心模块包括
- http接口管理，支持导出 openapi 结构，支持将http 导出成 mcp-server 所需的yml 格式，支持接口的版本记录和回退
- mcp-server 管理，支持管理 mcp-server 元信息， 支持选择多个 http 结构更新元信息，支持 mcp-server 的发布（当开发 MCP Server，编译为 Wasm 后动态加载到服务中）， 发布后具备 mcp-server 的完整功能，可以被client 链接使用，支持mcp-server 的版本记录和回退
- 路由管理，支持路由配置，比如 xxx/mcp-server/{name} 匹配 {name} 的 mcp-server



# references
https://higress.cn/ai/mcp-quick-start/?spm=36971b57.7beea2de.0.0.d85f20a9EauTak
https://higress.cn/ai/mcp-server/?spm=36971b57.7beea2de.0.0.d85f20a9TkuAxh
https://github.com/higress-group/gjson_template

Go语言与Gin框架 作为后端
vue3+tailwind 作为前端


mcp-server 定义激活完成之后，分三步
1. 提供这个mcp server 元信息接口，提供信息需要完整准确并且严格遵循 mcp 协议规范的接口元素，提供的接口信息通过该项目转发调用，其他AI应用通过获取元信息后，识别可用 tools，prompt 和 resource，提供给大模型
tools : https://modelcontextprotocol.io/specification/2025-03-26/server/tools
resources: https://modelcontextprotocol.io/specification/2025-03-26/server/resources
prompts: https://modelcontextprotocol.io/specification/2025-03-26/server/prompts
resources 和 prompts 先在元信息中保留

2. 提供一个完整的  mcp-server ，该服务严格遵循 mcp 协议，并提供所有相关接口， mcp-client 只需要通过 mcp-server/{server-name}接口就可以使用这个 server所有功能
3. 提供 mcp-client 的使用示例，支持 python，ts 语言，java 等常用语言
