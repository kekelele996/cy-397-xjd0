-- 合同模板生成与法律工单 API 服务初始化脚本（MySQL 8.0）
CREATE DATABASE IF NOT EXISTS contractapi CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE contractapi;
SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS contract_templates (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    category VARCHAR(64) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    content_html MEDIUMTEXT,
    variables JSON NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_contract_templates_code (code),
    KEY idx_contract_templates_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS contracts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    template_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(255) NOT NULL,
    content_text MEDIUMTEXT,
    content_html MEDIUMTEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    variables JSON NOT NULL,
    signed_at DATETIME(3) NULL,
    expires_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_contracts_user_id (user_id),
    KEY idx_contracts_template_id (template_id),
    KEY idx_contracts_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS contract_signers (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    contract_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(128) NOT NULL,
    role VARCHAR(64) NOT NULL,
    signed_at DATETIME(3) NULL,
    sign_info VARCHAR(512) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_contract_signers_contract_id (contract_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS legal_tickets (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    type VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    attachments JSON NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_legal_tickets_user_id (user_id),
    KEY idx_legal_tickets_type (type),
    KEY idx_legal_tickets_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ticket_replies (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_id BIGINT UNSIGNED NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'user',
    content TEXT NOT NULL,
    attachments JSON NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_ticket_replies_ticket_id (ticket_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS knowledge_faqs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    category VARCHAR(64) NOT NULL,
    question VARCHAR(512) NOT NULL,
    answer TEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_knowledge_faqs_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS template_favorites (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    template_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_template (user_id, template_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO contract_templates (code, name, category, description, content, content_html, variables) VALUES
('lease', '房屋租赁合同', 'lease', '用于住宅或商用房屋租赁的标准合同模板', '房屋租赁合同\n出租方（甲方）：{{.party_a}}\n承租方（乙方）：{{.party_b}}\n租赁房屋地址：{{.address}}\n月租金：人民币 {{.amount}} 元。\n租赁期限：自 {{.start_date}} 至 {{.end_date}}。\n签订日期：{{.sign_date}}', '<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><title>房屋租赁合同</title></head><body><h1>房屋租赁合同</h1><p>出租方（甲方）：{{.party_a}}</p><p>承租方（乙方）：{{.party_b}}</p><p>租赁房屋地址：{{.address}}</p><p>月租金：人民币 {{.amount}} 元。</p><p>租赁期限：自 {{.start_date}} 至 {{.end_date}}。</p><p>签订日期：{{.sign_date}}</p></body></html>', '[{"name":"party_a","label":"甲方","required":true},{"name":"party_b","label":"乙方","required":true},{"name":"address","label":"房屋地址","required":true},{"name":"amount","label":"金额","required":true},{"name":"start_date","label":"开始日期","required":true},{"name":"end_date","label":"结束日期","required":true},{"name":"sign_date","label":"签订日期","required":true}]'),
('labor', '劳动合同', 'labor', '用人单位与劳动者签订的劳动合同模板', '劳动合同\n用人单位（甲方）：{{.party_a}}\n劳动者（乙方）：{{.party_b}}\n岗位：{{.position}}\n月工资：人民币 {{.amount}} 元。\n合同期限：自 {{.start_date}} 至 {{.end_date}}。\n签订日期：{{.sign_date}}', '<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><title>劳动合同</title></head><body><h1>劳动合同</h1><p>用人单位（甲方）：{{.party_a}}</p><p>劳动者（乙方）：{{.party_b}}</p><p>岗位：{{.position}}</p><p>月工资：人民币 {{.amount}} 元。</p><p>合同期限：自 {{.start_date}} 至 {{.end_date}}。</p><p>签订日期：{{.sign_date}}</p></body></html>', '[{"name":"party_a","label":"甲方","required":true},{"name":"party_b","label":"乙方","required":true},{"name":"position","label":"岗位","required":true},{"name":"amount","label":"金额","required":true},{"name":"start_date","label":"开始日期","required":true},{"name":"end_date","label":"结束日期","required":true},{"name":"sign_date","label":"签订日期","required":true}]'),
('loan', '借款合同', 'loan', '自然人之间借款的标准合同模板', '借款合同\n出借人（甲方）：{{.party_a}}\n借款人（乙方）：{{.party_b}}\n借款金额：人民币 {{.amount}} 元。\n借款期限：自 {{.start_date}} 至 {{.end_date}}。\n签订日期：{{.sign_date}}', '<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><title>借款合同</title></head><body><h1>借款合同</h1><p>出借人（甲方）：{{.party_a}}</p><p>借款人（乙方）：{{.party_b}}</p><p>借款金额：人民币 {{.amount}} 元。</p><p>借款期限：自 {{.start_date}} 至 {{.end_date}}。</p><p>签订日期：{{.sign_date}}</p></body></html>', '[{"name":"party_a","label":"甲方","required":true},{"name":"party_b","label":"乙方","required":true},{"name":"amount","label":"金额","required":true},{"name":"start_date","label":"开始日期","required":true},{"name":"end_date","label":"结束日期","required":true},{"name":"sign_date","label":"签订日期","required":true}]'),
('cooperation', '合作协议', 'cooperation', '双方开展合作的通用协议模板', '合作协议\n甲方：{{.party_a}}\n乙方：{{.party_b}}\n合作事项：{{.subject}}\n合作期限：自 {{.start_date}} 至 {{.end_date}}。\n签订日期：{{.sign_date}}', '<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><title>合作协议</title></head><body><h1>合作协议</h1><p>甲方：{{.party_a}}</p><p>乙方：{{.party_b}}</p><p>合作事项：{{.subject}}</p><p>合作期限：自 {{.start_date}} 至 {{.end_date}}。</p><p>签订日期：{{.sign_date}}</p></body></html>', '[{"name":"party_a","label":"甲方","required":true},{"name":"party_b","label":"乙方","required":true},{"name":"subject","label":"合作事项","required":true},{"name":"start_date","label":"开始日期","required":true},{"name":"end_date","label":"结束日期","required":true},{"name":"sign_date","label":"签订日期","required":true}]'),
('nda', '保密协议', 'nda', '保护商业秘密与敏感信息的保密协议模板', '保密协议\n披露方（甲方）：{{.party_a}}\n接收方（乙方）：{{.party_b}}\n保密内容：{{.subject}}\n保密期限：自 {{.start_date}} 至 {{.end_date}}。\n签订日期：{{.sign_date}}', '<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><title>保密协议</title></head><body><h1>保密协议</h1><p>披露方（甲方）：{{.party_a}}</p><p>接收方（乙方）：{{.party_b}}</p><p>保密内容：{{.subject}}</p><p>保密期限：自 {{.start_date}} 至 {{.end_date}}。</p><p>签订日期：{{.sign_date}}</p></body></html>', '[{"name":"party_a","label":"甲方","required":true},{"name":"party_b","label":"乙方","required":true},{"name":"subject","label":"保密内容","required":true},{"name":"start_date","label":"开始日期","required":true},{"name":"end_date","label":"结束日期","required":true},{"name":"sign_date","label":"签订日期","required":true}]');

INSERT INTO knowledge_faqs (category, question, answer) VALUES
('劳动纠纷', '试用期最长可以约定多久？', '根据《劳动合同法》规定，劳动合同期限三个月以上不满一年的，试用期不得超过一个月；一年以上不满三年的，试用期不得超过二个月；三年以上固定期限和无固定期限的劳动合同，试用期不得超过六个月。'),
('合同纠纷', '合同没有约定违约金，违约了怎么办？', '即使合同未约定违约金，守约方仍可要求违约方赔偿实际损失，但需要提供损失证据。建议在合同中明确违约金计算方式以减少争议。'),
('房产纠纷', '租房合同未到期，房东要提前收回房屋怎么办？', '房东无正当理由提前收回房屋属于违约，承租人可以要求继续履行合同，或解除合同并要求退还押金、赔偿损失。'),
('知识产权', '员工在职期间完成的作品归谁？', '一般职务作品在单位物质技术条件下完成且属于单位业务范围的，权利归单位；双方另有约定的，从其约定。'),
('合同纠纷', '电子合同是否具有法律效力？', '可靠的电子签名与手写签名或盖章具有同等法律效力。签署电子合同时应使用合规的第三方电子签名平台并保存完整签署记录。'),
('其他', '如何保存证据更有效？', '优先保存原始载体、连贯的沟通记录、转账凭证等，并进行公证或使用可信时间戳固定证据。');
