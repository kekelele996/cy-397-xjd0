-- 000002_contract_ticket_link.sql
-- 打通合同库与纠纷工单：
-- 1) contracts 增加业务合同编号 contract_no（格式 HT-YYYY-000001）
-- 2) legal_tickets 增加登记合同编号与登记时合同状态快照
-- 3) 合同签署/过期联动的未关工单置为 review_pending（待复核/挂起）
-- 4) 旧工单按问题描述中的合同编号回填，回填不出的进入 ticket_contract_backfails 异常清单
--
-- 适用：MySQL 8.0（依赖 REGEXP_SUBSTR / REGEXP_LIKE，8.0.11+）
USE contractapi;
SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 1. 合同业务编号
ALTER TABLE contracts
    ADD COLUMN contract_no VARCHAR(32) NULL AFTER id;

-- 存量合同按创建年份 + 主键序号补齐编号
UPDATE contracts
SET contract_no = CONCAT('HT-', YEAR(created_at), '-', LPAD(id, 6, '0'))
WHERE contract_no IS NULL OR contract_no = '';

ALTER TABLE contracts
    MODIFY COLUMN contract_no VARCHAR(32) NOT NULL,
    ADD UNIQUE KEY uk_contracts_contract_no (contract_no);

-- 2. 工单登记的合同编号与合同状态快照
ALTER TABLE legal_tickets
    ADD COLUMN contract_no VARCHAR(32) NULL AFTER status,
    ADD COLUMN contract_status_snapshot VARCHAR(32) NULL AFTER contract_no,
    ADD KEY idx_legal_tickets_contract_no (contract_no);

-- 3. 旧数据回填：从问题描述提取编号并按合同库现状写入快照
UPDATE legal_tickets t
JOIN contracts c
  ON c.contract_no = REGEXP_SUBSTR(t.description, 'HT-[0-9]{4}-[0-9]{6}')
SET t.contract_no = c.contract_no,
    t.contract_status_snapshot = c.status
WHERE (t.contract_no IS NULL OR t.contract_no = '');

-- 4. 回填不出的工单单列到异常清单
CREATE TABLE IF NOT EXISTS ticket_contract_backfails (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_id BIGINT UNSIGNED NOT NULL,
    reason VARCHAR(255) NOT NULL,
    extracted_no VARCHAR(32) NOT NULL DEFAULT '',
    resolved TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_ticket_contract_backfails_ticket_id (ticket_id),
    KEY idx_ticket_contract_backfails_resolved (resolved)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4.1 描述里根本没有编号
INSERT INTO ticket_contract_backfails (ticket_id, reason, extracted_no)
SELECT t.id, 'no_contract_no_in_description', ''
FROM legal_tickets t
WHERE (t.contract_no IS NULL OR t.contract_no = '')
  AND t.description NOT REGEXP 'HT-[0-9]{4}-[0-9]{6}'
ON DUPLICATE KEY UPDATE reason = VALUES(reason), extracted_no = VALUES(extracted_no), resolved = 0;

-- 4.2 描述里有编号但合同库不存在
INSERT INTO ticket_contract_backfails (ticket_id, reason, extracted_no)
SELECT t.id, 'contract_not_found', REGEXP_SUBSTR(t.description, 'HT-[0-9]{4}-[0-9]{6}')
FROM legal_tickets t
WHERE (t.contract_no IS NULL OR t.contract_no = '')
  AND t.description REGEXP 'HT-[0-9]{4}-[0-9]{6}'
  AND NOT EXISTS (
      SELECT 1 FROM contracts c
      WHERE c.contract_no = REGEXP_SUBSTR(t.description, 'HT-[0-9]{4}-[0-9]{6}')
  )
ON DUPLICATE KEY UPDATE reason = VALUES(reason), extracted_no = VALUES(extracted_no), resolved = 0;
