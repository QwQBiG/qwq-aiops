package docs

// OpenAPISpec OpenAPI 规范
const OpenAPISpec = `
openapi: 3.0.0
info:
  title: QWQ AIOps Platform API
  description: 智能运维管理平台 REST API
  version: 1.0.0
  contact:
    name: QWQ Team
    email: support@qwq.io

servers:
  - url: http://localhost:8080
    description: 开发环境
  - url: https://api.qwq.io
    description: 生产环境

tags:
  - name: Websites
    description: 网站管理
  - name: SSL
    description: SSL 证书管理
  - name: DNS
    description: DNS 管理
  - name: Databases
    description: 数据库管理
  - name: Backups
    description: 备份恢复
  - name: Webhooks
    description: Webhook 和事件

paths:
  /api/v1/websites:
    get:
      tags: [Websites]
      summary: 列出网站
      parameters:
        - name: page
          in: query
          schema:
            type: integer
            default: 1
        - name: pageSize
          in: query
          schema:
            type: integer
            default: 20
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: object
                properties:
                  websites:
                    type: array
                    items:
                      $ref: '#/components/schemas/Website'
                  total:
                    type: integer
    post:
      tags: [Websites]
      summary: 创建网站
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Website'
      responses:
        '201':
          description: 创建成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Website'

  /api/v1/websites/{id}:
    get:
      tags: [Websites]
      summary: 获取网站详情
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Website'
    put:
      tags: [Websites]
      summary: 更新网站
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Website'
      responses:
        '200':
          description: 更新成功
    delete:
      tags: [Websites]
      summary: 删除网站
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: 删除成功

  /api/v1/ssl/certs:
    get:
      tags: [SSL]
      summary: 列出 SSL 证书
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/SSLCert'
    post:
      tags: [SSL]
      summary: 创建 SSL 证书记录
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SSLCert'
      responses:
        '201':
          description: 创建成功

  /api/v1/ssl/certs/request:
    post:
      tags: [SSL]
      summary: 申请 SSL 证书
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                domain:
                  type: string
                email:
                  type: string
                provider:
                  type: string
                  enum: [letsencrypt, self_signed, manual]
      responses:
        '201':
          description: 申请成功

  /api/v1/databases/connections:
    get:
      tags: [Databases]
      summary: 列出数据库连接
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/DatabaseConnection'
    post:
      tags: [Databases]
      summary: 创建数据库连接
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DatabaseConnection'
      responses:
        '201':
          description: 创建成功

  /api/v1/databases/query:
    post:
      tags: [Databases]
      summary: 执行 SQL 查询
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                connection_id:
                  type: integer
                sql:
                  type: string
                timeout:
                  type: integer
                max_rows:
                  type: integer
      responses:
        '200':
          description: 查询成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/QueryResult'

  /api/v1/backups/policies:
    get:
      tags: [Backups]
      summary: 列出备份策略
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/BackupPolicy'
    post:
      tags: [Backups]
      summary: 创建备份策略
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/BackupPolicy'
      responses:
        '201':
          description: 创建成功

  /api/v1/backups/policies/{id}/execute:
    post:
      tags: [Backups]
      summary: 执行备份
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '202':
          description: 备份任务已启动

  /api/v1/webhooks:
    get:
      tags: [Webhooks]
      summary: 列出 Webhooks
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Webhook'
    post:
      tags: [Webhooks]
      summary: 创建 Webhook
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Webhook'
      responses:
        '201':
          description: 创建成功

components:
  schemas:
    Website:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        domain:
          type: string
        ssl_enabled:
          type: boolean
        status:
          type: string
          enum: [active, inactive, error]
        created_at:
          type: string
          format: date-time

    SSLCert:
      type: object
      properties:
        id:
          type: integer
        domain:
          type: string
        provider:
          type: string
          enum: [letsencrypt, self_signed, manual]
        status:
          type: string
          enum: [valid, expired, pending, error]
        expiry_date:
          type: string
          format: date-time

    DatabaseConnection:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        type:
          type: string
          enum: [mysql, postgresql, redis, mongodb]
        host:
          type: string
        port:
          type: integer
        database:
          type: string
        status:
          type: string
          enum: [connected, disconnected, error]

    QueryResult:
      type: object
      properties:
        columns:
          type: array
          items:
            type: string
        rows:
          type: array
          items:
            type: object
        rows_affected:
          type: integer
        execution_time:
          type: number

    BackupPolicy:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        type:
          type: string
          enum: [database, files, container, system]
        schedule:
          type: string
        enabled:
          type: boolean
        storage_type:
          type: string
          enum: [local, s3, ftp, sftp]

    Webhook:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        url:
          type: string
        events:
          type: array
          items:
            type: string
        enabled:
          type: boolean

  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

security:
  - bearerAuth: []
`
