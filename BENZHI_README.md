# WaterNet 智慧供水管网调度平台

WaterNet 是城市供水管网的生产调度与过程控制平台。调度中心通过它实时掌握
泵站、清水池、泵组、阀门与压力分区的运行状态，并按压力目标自动决策泵组启停、
阀门开度与检修切换；所有指令、采样、执行回执与操作审计均落盘到本地文件。

## 构建

```bash
go build -mod=vendor ./...
```

## 运行

```bash
./waternet -data ./data -addr 127.0.0.1:8080
```

## 关键 API

- `GET /healthz` 健康检查
- `GET /api/v1/namespaces` 分区命名空间
- `GET /api/v1/stations/{id}` 泵站与清水池状态
- `GET /api/v1/pumps` 泵组运行概览
- `GET /api/v1/valves` 阀门开度与分区
- `GET /api/v1/audit` 操作审计
- `GET /api/v1/state` 全网状态快照

数据目录下自动生成 `state.json`、`readings.jsonl`、`audit.jsonl`、
`executions.jsonl` 与 `pumps/`、`valves/`、`stations/`、`snapshots/` 等
文件型持久化内容。
