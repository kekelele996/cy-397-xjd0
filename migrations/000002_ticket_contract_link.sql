-- 工单-合同按合同编号接通：
-- 1) legal_tickets 增加合同编号与建单时登记的合同状态/签署方快照；
-- 2) 旧数据按问题描述中的合同编号尽力回填（Go 回填命令/接口支持更多写法，
--    并把回填不出的工单单列；见 POST /api/v1/admin/tickets/backfill-contract）。
USE contractapi;

ALTER TABLE legal_tickets
    ADD COLUMN contract_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER status,
    ADD COLUMN contract_status_snapshot VARCHAR(32) NOT NULL DEFAULT '' AFTER contract_id,
    ADD COLUMN contract_signers_snapshot JSON NULL AFTER contract_status_snapshot,
    ADD KEY idx_legal_tickets_contract_id (contract_id);

-- 旧数据回填：识别「合同编号/合同号/合同编码」后紧跟的数字，按主键命中合同库则回填，
-- 并把该合同当前状态与签署方固化为工单快照。识别不出或合同不存在的行保持 contract_id = 0。
UPDATE legal_tickets t
JOIN (
    SELECT
        id,
        CAST(
            REGEXP_SUBSTR(
                REGEXP_SUBSTR(description, '合同(编号|号|编码)[[:space:]:：#]{0,3}[0-9]+'),
                '[0-9]+'
            ) AS UNSIGNED
        ) AS parsed_contract_id
    FROM legal_tickets
    WHERE contract_id = 0
) AS parsed ON parsed.id = t.id
JOIN contracts c ON c.id = parsed.parsed_contract_id
SET
    t.contract_id = c.id,
    t.contract_status_snapshot = c.status,
    t.contract_signers_snapshot = COALESCE((
        SELECT CONCAT('[', GROUP_CONCAT(JSON_OBJECT('name', s.name, 'role', s.role) ORDER BY s.role, s.name SEPARATOR ','), ']')
        FROM contract_signers s
        WHERE s.contract_id = c.id
    ), '[]')
WHERE t.contract_id = 0
  AND parsed.parsed_contract_id IS NOT NULL
  AND parsed.parsed_contract_id > 0;

-- 回填不出的旧工单（人工核对清单）：
-- SELECT id, user_id, title, description FROM legal_tickets WHERE contract_id = 0 ORDER BY id;
