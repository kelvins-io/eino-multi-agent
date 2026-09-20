---
name: research
description: 针对一个主题检索公开信息，比较来源并生成研究报告
---

# 调研

适用：行业研究、竞品对比、资料综述。

## 流程

1. 把问题拆成 3-6 个可检索子问题。
2. 使用 web_search 检索，再用 fetch_url 阅读关键页面。
3. 记录来源 URL，不要编造数据。
4. 将结论写入 `output/research-report.md`，包含：
   - 问题与范围
   - 发现
   - 证据与来源
   - 不确定性和后续建议
