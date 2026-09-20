---
name: weekly-report
description: 根据表格或文本资料生成结构化周报，并写入 output/
---

# 周报生成

适用：用户上传 CSV/Excel/Markdown，要求生成本周工作或销售周报。

## 流程

1. 列出 `input/` 中的文件并读取内容。
2. 识别时间范围、核心指标、异常点和结论。
3. 若是表格，交给 office 子代理做汇总计算。
4. 在 `output/weekly-report.md` 输出周报，建议结构：
   - 本周概览
   - 关键指标
   - 亮点与风险
   - 下周建议
5. 如有明细，额外输出 `output/weekly-metrics.csv`。
