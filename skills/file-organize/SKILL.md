---
name: file-organize
description: 整理工作区内的文件，按规则重命名或归类，并把结果放到 output/
---

# 文件整理

适用：批量整理上传资料、统一命名、生成目录。

## 流程

1. 列出 input/ 全部文件。
2. 根据文件名、扩展名和内容做分类。
3. 不要删除用户原始文件。需要新结构时复制到 output/organized/。
4. 输出 `output/file-index.md` 说明分类规则和最终路径。
5. 覆盖或大规模改写前先调用 confirm_action。
