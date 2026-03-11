import React, { useState, useMemo } from 'react';
import { Menu, Lock, ChevronRight, Server, FileJson, Shield, CheckCircle, AlertTriangle, Info, Play, Send, RefreshCw } from 'lucide-react';

// The provided OpenAPI Specification
const apiSpec = {
  "schemes": ["http"],
  "swagger": "2.0",
  "info": {
    "description": "Forum backend API.",
    "title": "Forum API",
    "contact": {},
    "version": "1.0"
  },
  "basePath": "/api",
  "paths": {
    "/auth/login": {
      "post": {
        "description": "Authenticate with email and password, return an auth token.",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "tags": ["Auth"],
        "summary": "Login",
        "parameters": [
          {
            "description": "Login payload",
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": { "$ref": "#/definitions/models.LoginDTO" }
          }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.AuthResponse" } },
          "400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "401": { "description": "Unauthorized", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    },
    "/auth/register": {
      "post": {
        "description": "Create a new account and return an auth token.",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "tags": ["Auth"],
        "summary": "Register a new user",
        "parameters": [
          {
            "description": "Registration payload",
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": { "$ref": "#/definitions/models.RegisterDTO" }
          }
        ],
        "responses": {
          "201": { "description": "Created", "schema": { "$ref": "#/definitions/models.AuthResponse" } },
          "400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "409": { "description": "Conflict", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    },
    "/posts": {
      "get": {
        "security": [{ "BearerAuth": [] }],
        "description": "Get a paginated list of posts.",
        "produces": ["application/json"],
        "tags": ["Posts"],
        "summary": "List posts",
        "parameters": [
          { "type": "integer", "default": 1, "description": "Page number", "name": "page", "in": "query" },
          { "type": "integer", "default": 10, "description": "Page size", "name": "limit", "in": "query" }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.PostListResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      },
      "post": {
        "security": [{ "BearerAuth": [] }],
        "description": "Create a new post.",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "tags": ["Posts"],
        "summary": "Create a post",
        "parameters": [
          {
            "description": "Post payload",
            "name": "payload",
            "in": "body",
            "required": true,
            "schema": { "$ref": "#/definitions/models.CreatePostDTO" }
          }
        ],
        "responses": {
          "201": { "description": "Created", "schema": { "$ref": "#/definitions/models.Post" } },
          "400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    },
    "/posts/{id}": {
      "get": {
        "security": [{ "BearerAuth": [] }],
        "description": "Get a single post by ID.",
        "produces": ["application/json"],
        "tags": ["Posts"],
        "summary": "Get a post",
        "parameters": [
          { "type": "string", "description": "Post ID", "name": "id", "in": "path", "required": true }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.Post" } },
          "404": { "description": "Not Found", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      },
      "put": {
        "security": [{ "BearerAuth": [] }],
        "description": "Update an existing post by ID.",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "tags": ["Posts"],
        "summary": "Update a post",
        "parameters": [
          { "type": "string", "description": "Post ID", "name": "id", "in": "path", "required": true },
          { "description": "Update payload", "name": "payload", "in": "body", "required": true, "schema": { "$ref": "#/definitions/models.UpdatePostDTO" } }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.MessageResponse" } },
          "400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "404": { "description": "Not Found", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      },
      "delete": {
        "security": [{ "BearerAuth": [] }],
        "description": "Delete a post by ID.",
        "produces": ["application/json"],
        "tags": ["Posts"],
        "summary": "Delete a post",
        "parameters": [
          { "type": "string", "description": "Post ID", "name": "id", "in": "path", "required": true }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.MessageResponse" } },
          "403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "404": { "description": "Not Found", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    },
    "/posts/{id}/comments": {
      "post": {
        "security": [{ "BearerAuth": [] }],
        "description": "Add a comment to a post.",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "tags": ["Comments"],
        "summary": "Add a comment",
        "parameters": [
          { "type": "string", "description": "Post ID", "name": "id", "in": "path", "required": true },
          { "description": "Comment payload", "name": "payload", "in": "body", "required": true, "schema": { "$ref": "#/definitions/models.AddCommandDTO" } }
        ],
        "responses": {
          "201": { "description": "Created", "schema": { "$ref": "#/definitions/models.MessageResponse" } },
          "400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "404": { "description": "Not Found", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    },
    "/posts/{id}/comments/{commentId}": {
      "delete": {
        "security": [{ "BearerAuth": [] }],
        "description": "Delete a comment by ID.",
        "produces": ["application/json"],
        "tags": ["Comments"],
        "summary": "Delete a comment",
        "parameters": [
          { "type": "string", "description": "Post ID", "name": "id", "in": "path", "required": true },
          { "type": "string", "description": "Comment ID", "name": "commentId", "in": "path", "required": true }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.MessageResponse" } },
          "403": { "description": "Forbidden", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "404": { "description": "Not Found", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    },
    "/posts/{id}/comments/{commentId}/vote": {
      "post": {
        "security": [{ "BearerAuth": [] }],
        "description": "Upvote or downvote a comment.",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "tags": ["Votes"],
        "summary": "Vote on a comment",
        "parameters": [
          { "type": "string", "description": "Post ID", "name": "id", "in": "path", "required": true },
          { "type": "string", "description": "Comment ID", "name": "commentId", "in": "path", "required": true },
          { "description": "Vote payload", "name": "payload", "in": "body", "required": true, "schema": { "$ref": "#/definitions/models.VoteDTO" } }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.MessageResponse" } },
          "400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "404": { "description": "Not Found", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    },
    "/posts/{id}/vote": {
      "post": {
        "security": [{ "BearerAuth": [] }],
        "description": "Upvote or downvote a post.",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "tags": ["Votes"],
        "summary": "Vote on a post",
        "parameters": [
          { "type": "string", "description": "Post ID", "name": "id", "in": "path", "required": true },
          { "description": "Vote payload", "name": "payload", "in": "body", "required": true, "schema": { "$ref": "#/definitions/models.VoteDTO" } }
        ],
        "responses": {
          "200": { "description": "OK", "schema": { "$ref": "#/definitions/models.MessageResponse" } },
          "400": { "description": "Bad Request", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "404": { "description": "Not Found", "schema": { "$ref": "#/definitions/models.ErrorResponse" } },
          "500": { "description": "Internal Server Error", "schema": { "$ref": "#/definitions/models.ErrorResponse" } }
        }
      }
    }
  },
  "definitions": {
    "models.AddCommandDTO": { "type": "object", "required": ["author", "content", "postId"], "properties": { "author": { "type": "string" }, "content": { "type": "string" }, "custom": { "type": "object", "additionalProperties": {} }, "postId": { "type": "string" }, "replyToId": { "type": "string" } } },
    "models.AuthResponse": { "type": "object", "properties": { "token": { "type": "string" }, "user": { "$ref": "#/definitions/models.User" } } },
    "models.Comment": { "type": "object", "properties": { "author": { "type": "string" }, "content": { "type": "string" }, "createdAt": { "type": "string" }, "custom": { "type": "object", "additionalProperties": {} }, "downVotes": { "type": "integer" }, "id": { "type": "string" }, "replyToId": { "type": "string" }, "upVotes": { "type": "integer" } } },
    "models.CreatePostDTO": { "type": "object", "required": ["author", "content", "title"], "properties": { "author": { "type": "string" }, "content": { "type": "string" }, "custom": { "type": "object", "additionalProperties": {} }, "title": { "type": "string" } } },
    "models.ErrorResponse": { "type": "object", "properties": { "error": { "type": "string" } } },
    "models.LoginDTO": { "type": "object", "required": ["email", "password"], "properties": { "email": { "type": "string" }, "password": { "type": "string" } } },
    "models.MessageResponse": { "type": "object", "properties": { "message": { "type": "string" } } },
    "models.Post": { "type": "object", "properties": { "author": { "type": "string" }, "comments": { "type": "array", "items": { "$ref": "#/definitions/models.Comment" } }, "content": { "type": "string" }, "createdAt": { "type": "string" }, "custom": { "type": "object", "additionalProperties": {} }, "downVotes": { "type": "integer" }, "id": { "type": "string" }, "title": { "type": "string" }, "upVotes": { "type": "integer" }, "updatedAt": { "type": "string" } } },
    "models.PostListResponse": { "type": "object", "properties": { "limit": { "type": "integer" }, "page": { "type": "integer" }, "posts": { "type": "array", "items": { "$ref": "#/definitions/models.Post" } } } },
    "models.RegisterDTO": { "type": "object", "required": ["email", "password", "username"], "properties": { "email": { "type": "string" }, "password": { "type": "string", "minLength": 6 }, "username": { "type": "string", "maxLength": 20, "minLength": 3 } } },
    "models.UpdatePostDTO": { "type": "object", "properties": { "content": { "type": "string" }, "custom": { "type": "object", "additionalProperties": {} }, "title": { "type": "string" } } },
    "models.User": { "type": "object", "properties": { "createdAt": { "type": "string" }, "email": { "type": "string" }, "id": { "type": "string" }, "username": { "type": "string" } } },
    "models.VoteDTO": { "type": "object", "required": ["action"], "properties": { "action": { "description": "只能是 up 或 down", "type": "string", "enum": ["up", "down"] } } }
  },
  "securityDefinitions": {
    "BearerAuth": { "type": "apiKey", "name": "Authorization", "in": "header" }
  }
};

const MethodBadge = ({ method }) => {
  const colors = {
    get: 'bg-blue-100 text-blue-700 border-blue-200',
    post: 'bg-emerald-100 text-emerald-700 border-emerald-200',
    put: 'bg-amber-100 text-amber-700 border-amber-200',
    delete: 'bg-rose-100 text-rose-700 border-rose-200',
    patch: 'bg-purple-100 text-purple-700 border-purple-200',
  };
  return (
    <span className={`px-2 py-1 uppercase text-xs font-bold border rounded-md mr-3 ${colors[method] || 'bg-gray-100 text-gray-700'}`}>
      {method}
    </span>
  );
};

const resolveRef = (ref, definitions) => {
  if (!ref) return null;
  const refName = ref.split('/').pop();
  return definitions[refName] || null;
};

// 自动根据 Schema 生成 Mock 假数据填充请求体
const generateMockData = (schema, definitions) => {
  if (!schema) return null;
  if (schema.$ref) return generateMockData(resolveRef(schema.$ref, definitions), definitions);
  if (schema.type === 'object' || schema.properties) {
    const result = {};
    Object.entries(schema.properties || {}).forEach(([k, v]) => {
      result[k] = generateMockData(v, definitions);
    });
    return result;
  }
  if (schema.type === 'array') return schema.items ? [generateMockData(schema.items, definitions)] : [];
  if (schema.type === 'string') return schema.enum ? schema.enum[0] : "string";
  if (schema.type === 'integer' || schema.type === 'number') return 0;
  if (schema.type === 'boolean') return true;
  return "";
};

const renderSchemaType = (schema, definitions, isRequired = false) => {
  if (!schema) return <span className="text-gray-500">any</span>;
  if (schema.$ref) {
    const refName = schema.$ref.split('/').pop();
    return <span className="text-indigo-600 font-medium">{refName}</span>;
  }
  if (schema.type === 'array' && schema.items) {
    return <span>Array&lt;{renderSchemaType(schema.items, definitions)}&gt;</span>;
  }
  return <span className="text-pink-600 font-mono text-sm">{schema.type || 'object'}</span>;
};

const SchemaTable = ({ schemaRef, definitions }) => {
  const schema = resolveRef(schemaRef, definitions);
  if (!schema) return <div className="text-gray-500 italic text-sm">No schema defined</div>;

  const requiredProps = schema.required || [];

  return (
    <div className="border border-slate-200 rounded-lg overflow-hidden bg-white mt-3">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50">
          <tr>
            <th className="px-4 py-3 text-left font-semibold text-slate-700 w-1/4">Property</th>
            <th className="px-4 py-3 text-left font-semibold text-slate-700 w-1/4">Type</th>
            <th className="px-4 py-3 text-left font-semibold text-slate-700">Description / Constraints</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {Object.entries(schema.properties || {}).map(([key, prop]) => {
            const isReq = requiredProps.includes(key);
            return (
              <tr key={key} className="hover:bg-slate-50 transition-colors">
                <td className="px-4 py-3 align-top">
                  <div className="font-mono text-slate-800">{key}</div>
                  {isReq && <div className="text-xs text-rose-500 mt-1 font-semibold">required</div>}
                </td>
                <td className="px-4 py-3 align-top">
                  {renderSchemaType(prop, definitions)}
                </td>
                <td className="px-4 py-3 align-top text-slate-600">
                  {prop.description && <div className="mb-1">{prop.description}</div>}
                  {prop.enum && <div><span className="font-semibold text-xs">Enum: </span>{prop.enum.join(', ')}</div>}
                  {prop.minLength !== undefined && <div><span className="font-semibold text-xs">Min Length: </span>{prop.minLength}</div>}
                  {prop.maxLength !== undefined && <div><span className="font-semibold text-xs">Max Length: </span>{prop.maxLength}</div>}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

// 新增组件：API 接口在线测试器
const EndpointTester = ({ endpoint, apiSpec, globalToken, setGlobalToken, baseUrl, setBaseUrl }) => {
  const [pathParams, setPathParams] = useState({});
  const [queryParams, setQueryParams] = useState({});
  const [requestBody, setRequestBody] = useState('');
  const [response, setResponse] = useState(null);
  const [loading, setLoading] = useState(false);

  // 每次切换接口时，重置并初始化默认参数与 Body
  React.useEffect(() => {
    const initialPaths = {};
    const initialQueries = {};
    let initialBody = '';

    (endpoint.parameters || []).forEach(p => {
      if (p.in === 'path') initialPaths[p.name] = p.default || '';
      if (p.in === 'query') initialQueries[p.name] = p.default || '';
      if (p.in === 'body' && p.schema) {
        initialBody = JSON.stringify(generateMockData(p.schema, apiSpec.definitions), null, 2);
      }
    });

    setPathParams(initialPaths);
    setQueryParams(initialQueries);
    setRequestBody(initialBody);
    setResponse(null);
  }, [endpoint, apiSpec]);

  const handleSend = async () => {
    setLoading(true);
    setResponse(null);

    try {
      // 构建最终的 URL (处理路径参数)
      let finalUrl = baseUrl + endpoint.path;
      Object.entries(pathParams).forEach(([k, v]) => {
        finalUrl = finalUrl.replace(`{${k}}`, encodeURIComponent(v));
      });

      // 拼接 Query 参数
      const q = new URLSearchParams();
      Object.entries(queryParams).forEach(([k, v]) => {
        if (v !== '') q.append(k, v);
      });
      const queryString = q.toString();
      if (queryString) finalUrl += `?${queryString}`;

      // 构造请求头
      const headers = {};
      if (endpoint.consumes?.includes('application/json')) {
        headers['Content-Type'] = 'application/json';
      }
      if (endpoint.security && globalToken) {
        headers['Authorization'] = `Bearer ${globalToken}`;
      }

      const options = {
        method: endpoint.method.toUpperCase(),
        headers
      };

      // 如果有 body 参数则带上
      if (endpoint.parameters?.some(p => p.in === 'body') && requestBody) {
        options.body = requestBody;
      }

      const startTime = Date.now();
      const res = await fetch(finalUrl, options);
      const time = Date.now() - startTime;

      let data;
      const contentType = res.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        data = await res.json();
      } else {
        data = await res.text();
      }

      setResponse({
        status: res.status,
        statusText: res.statusText,
        time,
        data
      });

      // 自动保存 Token 以供后续请求无缝使用
      if (res.status === 200 && endpoint.path === '/auth/login' && data?.token) {
        setGlobalToken(data.token);
      }
      if (res.status === 201 && endpoint.path === '/auth/register' && data?.token) {
        setGlobalToken(data.token);
      }

    } catch (err) {
      setResponse({
        status: 'Error',
        statusText: 'Network Error / CORS Issue',
        time: 0,
        data: err.message + '\n\n注: 浏览器默认拦截跨域请求 (CORS)。如果您在测试外部 API，请确保服务器开启了跨域支持。'
      });
    } finally {
      setLoading(false);
    }
  };

  const hasPathParams = endpoint.parameters?.some(p => p.in === 'path');
  const hasQueryParams = endpoint.parameters?.some(p => p.in === 'query');
  const hasBody = endpoint.parameters?.some(p => p.in === 'body');
  const requiresAuth = !!endpoint.security;

  return (
    <div className="space-y-6">
      <div className="bg-slate-50 p-5 rounded-lg border border-slate-200">
        <div className="mb-4">
          <label className="block text-sm font-semibold text-slate-700 mb-1">Base URL (API Host)</label>
          <input type="text" value={baseUrl} onChange={e => setBaseUrl(e.target.value)} className="w-full border-slate-300 rounded-md shadow-sm text-sm px-3 py-2 border focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" />
        </div>
        {requiresAuth && (
          <div>
            <label className="block text-sm font-semibold text-slate-700 mb-1 flex items-center gap-2">
              <Lock className="w-4 h-4 text-amber-500" /> Bearer Token (自动携带)
            </label>
            <input type="text" value={globalToken} onChange={e => setGlobalToken(e.target.value)} placeholder="在此粘贴您的 Token... (或先调用 Login 接口自动获取)" className="w-full border-slate-300 rounded-md shadow-sm text-sm px-3 py-2 border focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-amber-500" />
          </div>
        )}
      </div>

      {(hasPathParams || hasQueryParams) && (
        <div>
          <h4 className="text-lg font-bold text-slate-800 mb-3 border-b pb-2">请求参数 (Parameters)</h4>
          <div className="space-y-4">
            {endpoint.parameters.filter(p => p.in === 'path' || p.in === 'query').map(p => (
              <div key={p.name} className="flex flex-col sm:flex-row sm:items-center gap-2">
                <label className="sm:w-1/3 text-sm font-medium text-slate-700 font-mono">
                  {p.name} {p.required && <span className="text-rose-500">*</span>}
                  <span className="ml-2 text-xs text-slate-400">({p.in})</span>
                </label>
                <input
                  type="text"
                  value={p.in === 'path' ? (pathParams[p.name] || '') : (queryParams[p.name] || '')}
                  onChange={e => p.in === 'path'
                    ? setPathParams({ ...pathParams, [p.name]: e.target.value })
                    : setQueryParams({ ...queryParams, [p.name]: e.target.value })}
                  className="flex-1 border-slate-300 rounded-md shadow-sm text-sm px-3 py-2 border focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  placeholder={p.description || `输入 ${p.name}`}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {hasBody && (
        <div>
          <h4 className="text-lg font-bold text-slate-800 mb-3 border-b pb-2">请求体 (Request Body JSON)</h4>
          <textarea
            value={requestBody}
            onChange={e => setRequestBody(e.target.value)}
            rows={8}
            className="w-full font-mono text-sm border-slate-300 rounded-md shadow-sm px-3 py-2 border focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
      )}

      <div className="pt-4 border-t border-slate-100">
        <button
          onClick={handleSend}
          disabled={loading}
          className="flex items-center gap-2 bg-indigo-600 text-white px-5 py-2.5 rounded-lg font-semibold hover:bg-indigo-700 transition-colors disabled:opacity-50"
        >
          {loading ? <RefreshCw className="w-5 h-5 animate-spin" /> : <Send className="w-5 h-5" />}
          发送请求
        </button>
      </div>

      {response && (
        <div className="mt-8 border rounded-lg overflow-hidden shadow-sm">
          <div className={`px-4 py-3 border-b flex items-center justify-between ${response.status >= 200 && response.status < 300 ? 'bg-emerald-50 border-emerald-200 text-emerald-800' : 'bg-rose-50 border-rose-200 text-rose-800'}`}>
            <div className="font-bold">
              状态码: {response.status} {response.statusText}
            </div>
            <div className="text-sm font-medium">
              耗时: {response.time}ms
            </div>
          </div>
          <div className="p-4 bg-slate-900 overflow-x-auto">
            <pre className="text-emerald-400 font-mono text-sm whitespace-pre-wrap">
              {typeof response.data === 'object' ? JSON.stringify(response.data, null, 2) : response.data}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
};

export default function App() {
  const [selectedEndpoint, setSelectedEndpoint] = useState(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('docs');

  // 使用 localStorage 初始化并保存 Token 和 BaseUrl
  const [globalToken, setGlobalToken] = useState(() => localStorage.getItem('api_global_token') || '');
  const [baseUrl, setBaseUrl] = useState(() => localStorage.getItem('api_base_url') || 'http://localhost:3000/api');

  // 当 Token 改变时，自动存入 localStorage
  React.useEffect(() => {
    localStorage.setItem('api_global_token', globalToken);
  }, [globalToken]);

  // 当 Base URL 改变时，自动存入 localStorage
  React.useEffect(() => {
    localStorage.setItem('api_base_url', baseUrl);
  }, [baseUrl]);

  // Parse and group the endpoints
  const { endpoints, tags } = useMemo(() => {
    const grouped = {};
    const allEndpoints = [];

    Object.entries(apiSpec.paths).forEach(([path, methods]) => {
      Object.entries(methods).forEach(([method, details]) => {
        const tag = details.tags?.[0] || 'Default';
        if (!grouped[tag]) grouped[tag] = [];

        const endpointInfo = { id: `${method}-${path}`, path, method, ...details };
        grouped[tag].push(endpointInfo);
        allEndpoints.push(endpointInfo);
      });
    });

    return { endpoints: grouped, tags: Object.keys(grouped) };
  }, []);

  const activeItem = selectedEndpoint || (tags.length > 0 && endpoints[tags[0]]?.[0]);

  return (
    <div className="flex h-screen w-full bg-white font-sans text-slate-800 selection:bg-indigo-100 selection:text-indigo-900">

      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-slate-800/20 z-20 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside className={`
                fixed lg:static inset-y-0 left-0 z-30
                w-72 bg-slate-50 border-r border-slate-200 overflow-y-auto
                transform transition-transform duration-300 ease-in-out
                ${sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
            `}>
        <div className="p-6 pb-4 border-b border-slate-200 bg-white sticky top-0 z-10">
          <h1 className="text-xl font-bold text-slate-900 flex items-center gap-2">
            <Server className="w-5 h-5 text-indigo-600" />
            {apiSpec.info.title}
          </h1>
          <div className="flex items-center gap-2 mt-2 text-sm text-slate-500">
            <span className="bg-slate-100 px-2 py-0.5 rounded text-xs font-semibold">v{apiSpec.info.version}</span>
            <span>{apiSpec.basePath}</span>
          </div>
        </div>

        <div className="p-4">
          {tags.map(tag => (
            <div key={tag} className="mb-6">
              <h2 className="text-xs font-bold text-slate-400 uppercase tracking-wider mb-2 ml-2">{tag}</h2>
              <ul className="space-y-1">
                {endpoints[tag].map(endpoint => {
                  const isActive = activeItem?.id === endpoint.id;
                  return (
                    <li key={endpoint.id}>
                      <button
                        onClick={() => {
                          setSelectedEndpoint(endpoint);
                          setSidebarOpen(false);
                        }}
                        className={`
                                                    w-full text-left px-3 py-2 rounded-lg text-sm font-medium transition-all
                                                    flex items-center gap-3 group
                                                    ${isActive ? 'bg-indigo-50 text-indigo-700' : 'text-slate-600 hover:bg-slate-100'}
                                                `}
                      >
                        <span className={`
                                                    uppercase text-[10px] font-bold w-10 text-center
                                                    ${endpoint.method === 'get' ? 'text-blue-600' :
                            endpoint.method === 'post' ? 'text-emerald-600' :
                              endpoint.method === 'put' ? 'text-amber-600' :
                                'text-rose-600'}
                                                `}>
                          {endpoint.method}
                        </span>
                        <span className="truncate">{endpoint.path}</span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto bg-white relative">
        {/* Header for mobile */}
        <header className="lg:hidden sticky top-0 z-10 bg-white/80 backdrop-blur-md border-b border-slate-200 px-4 py-3 flex items-center gap-3">
          <button onClick={() => setSidebarOpen(true)} className="p-2 -ml-2 rounded-md hover:bg-slate-100">
            <Menu className="w-5 h-5 text-slate-600" />
          </button>
          <span className="font-semibold text-slate-800">{apiSpec.info.title}</span>
        </header>

        {activeItem ? (
          <div className="max-w-4xl mx-auto p-6 lg:p-10 pb-24">
            <div className="mb-8">
              <h2 className="text-3xl font-bold text-slate-900 mb-4 tracking-tight">{activeItem.summary || activeItem.description}</h2>
              <div className="flex flex-wrap items-center gap-4 bg-slate-50 border border-slate-200 rounded-lg p-4 shadow-sm">
                <MethodBadge method={activeItem.method} />
                <div className="font-mono text-slate-700 text-lg break-all">
                  <span className="text-slate-400 select-none">{apiSpec.basePath}</span>
                  <span className="font-semibold text-slate-900">{activeItem.path}</span>
                </div>
                {activeItem.security && (
                  <div className="ml-auto flex items-center gap-1.5 text-amber-600 bg-amber-50 px-2.5 py-1 rounded-md text-xs font-semibold border border-amber-200">
                    <Lock className="w-3.5 h-3.5" />
                    Bearer Auth
                  </div>
                )}
              </div>
              {activeItem.description && (
                <p className="mt-4 text-slate-600 leading-relaxed text-lg">
                  {activeItem.description}
                </p>
              )}
            </div>

            {/* TABS 标签页切换 */}
            <div className="border-b border-slate-200 mb-8 flex gap-6">
              <button
                onClick={() => setActiveTab('docs')}
                className={`pb-3 text-sm font-semibold border-b-2 transition-colors ${activeTab === 'docs' ? 'border-indigo-600 text-indigo-600' : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'}`}
              >
                文档详情 (Documentation)
              </button>
              <button
                onClick={() => setActiveTab('test')}
                className={`pb-3 text-sm font-semibold border-b-2 transition-colors flex items-center gap-1.5 ${activeTab === 'test' ? 'border-emerald-600 text-emerald-600' : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'}`}
              >
                <Play className="w-4 h-4" /> 在线测试 (Try it out)
              </button>
            </div>

            {activeTab === 'docs' ? (
              <>
                {/* Parameters */}
                {activeItem.parameters && activeItem.parameters.length > 0 && (
                  <div className="mb-12">
                    <h3 className="text-xl font-bold text-slate-800 border-b border-slate-200 pb-2 mb-4">Request</h3>

                    {/* Path / Query Params */}
                    {activeItem.parameters.filter(p => p.in !== 'body').length > 0 && (
                      <div className="mb-6">
                        <h4 className="text-sm font-semibold text-slate-500 uppercase tracking-wide mb-3">Parameters</h4>
                        <div className="bg-white border border-slate-200 rounded-lg overflow-hidden shadow-sm">
                          <table className="min-w-full divide-y divide-slate-200 text-sm">
                            <thead className="bg-slate-50">
                              <tr>
                                <th className="px-4 py-3 text-left font-semibold text-slate-700 w-1/4">Name</th>
                                <th className="px-4 py-3 text-left font-semibold text-slate-700 w-1/4">Location</th>
                                <th className="px-4 py-3 text-left font-semibold text-slate-700">Description</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-slate-100">
                              {activeItem.parameters.filter(p => p.in !== 'body').map(param => (
                                <tr key={param.name}>
                                  <td className="px-4 py-3 align-top">
                                    <span className="font-mono text-slate-800">{param.name}</span>
                                    {param.required && <span className="text-rose-500 text-xs font-bold ml-2">REQ</span>}
                                  </td>
                                  <td className="px-4 py-3 align-top">
                                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600">
                                      {param.in}
                                    </span>
                                  </td>
                                  <td className="px-4 py-3 align-top text-slate-600">
                                    <div>{param.description}</div>
                                    <div className="text-xs mt-1 text-slate-400">Type: <span className="font-mono">{param.type}</span></div>
                                    {param.default !== undefined && <div className="text-xs mt-1 text-slate-400">Default: <span className="font-mono text-slate-600">{param.default}</span></div>}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </div>
                    )}

                    {/* Body Payload */}
                    {activeItem.parameters.filter(p => p.in === 'body').map(param => (
                      <div key={param.name} className="mb-6">
                        <div className="flex items-center gap-2 mb-3">
                          <h4 className="text-sm font-semibold text-slate-500 uppercase tracking-wide">Request Body</h4>
                          <span className="bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded text-xs font-semibold">application/json</span>
                          {param.required && <span className="text-rose-500 text-xs font-bold ml-2">Required</span>}
                        </div>
                        <p className="text-sm text-slate-600 mb-3">{param.description}</p>
                        <SchemaTable schemaRef={param.schema?.$ref} definitions={apiSpec.definitions} />
                      </div>
                    ))}
                  </div>
                )}

                {/* Responses */}
                {activeItem.responses && (
                  <div>
                    <h3 className="text-xl font-bold text-slate-800 border-b border-slate-200 pb-2 mb-4">Responses</h3>
                    <div className="space-y-6">
                      {Object.entries(activeItem.responses).map(([code, response]) => {
                        const isSuccess = code.startsWith('2');
                        const isError = code.startsWith('4') || code.startsWith('5');

                        return (
                          <div key={code} className={`border rounded-lg overflow-hidden shadow-sm ${isSuccess ? 'border-emerald-200' : isError ? 'border-rose-200' : 'border-slate-200'}`}>
                            <div className={`px-4 py-3 flex justify-between items-center ${isSuccess ? 'bg-emerald-50' : isError ? 'bg-rose-50' : 'bg-slate-50'}`}>
                              <div className="flex items-center gap-3">
                                <span className={`font-mono text-lg font-bold ${isSuccess ? 'text-emerald-700' : isError ? 'text-rose-700' : 'text-slate-700'}`}>
                                  {code}
                                </span>
                                <span className={`text-sm font-medium ${isSuccess ? 'text-emerald-800' : isError ? 'text-rose-800' : 'text-slate-800'}`}>
                                  {response.description}
                                </span>
                              </div>
                            </div>

                            {response.schema && response.schema.$ref && (
                              <div className="p-4 bg-white">
                                <h5 className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3 flex items-center gap-2">
                                  <FileJson className="w-4 h-4" />
                                  Response Schema
                                </h5>
                                <SchemaTable schemaRef={response.schema.$ref} definitions={apiSpec.definitions} />
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
              </>
            ) : (
              <EndpointTester
                endpoint={activeItem}
                apiSpec={apiSpec}
                globalToken={globalToken}
                setGlobalToken={setGlobalToken}
                baseUrl={baseUrl}
                setBaseUrl={setBaseUrl}
              />
            )}
          </div>
        ) : (
          <div className="flex h-full items-center justify-center p-6 text-center">
            <div>
              <Server className="w-16 h-16 text-slate-200 mx-auto mb-4" />
              <h2 className="text-2xl font-bold text-slate-800 mb-2">Welcome to {apiSpec.info.title} Docs</h2>
              <p className="text-slate-500 max-w-md mx-auto">Select an endpoint from the sidebar to view its details, request parameters, and response schemas.</p>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
