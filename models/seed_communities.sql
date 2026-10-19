-- 社区初始化数据：请先完成建表，再手动导入本文件。
-- 相同ID或名称已存在时保留原记录，不覆盖现有社区。
-- 应用启动不会自动执行本文件。
SET NAMES utf8mb4;

INSERT IGNORE INTO community (community_id, community_name, introduction) VALUES
  (1, 'Go语言', '交流Go基础、并发编程、标准库与工程实践。'),
  (2, '后端开发', '讨论接口设计、Gin框架、登录鉴权与后端项目实践。'),
  (3, '数据库与缓存', '交流MySQL、Redis、SQL优化和数据存储经验。'),
  (4, '学习交流', '分享学习笔记、项目心得、面试经验与技术问题。');
