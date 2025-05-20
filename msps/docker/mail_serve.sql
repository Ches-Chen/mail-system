-- 创建数据库
CREATE DATABASE IF NOT EXISTS `mail_serve` COLLATE 'utf8mb4_general_ci';
USE `mail_serve`;

-- 用户表
CREATE TABLE `users` (
                         `id` bigint(20) NOT NULL AUTO_INCREMENT,
                         `username` varchar(50) NOT NULL,
                         `password` varchar(255) NOT NULL,
                         `phone` varchar(20) NOT NULL,
                         `role` enum('user', 'admin') NOT NULL DEFAULT 'user',
                         `status` enum('active', 'disabled') NOT NULL DEFAULT 'active',
                         `created_at` datetime(3) NULL DEFAULT NULL,
                         `updated_at` datetime(3) NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(3),
                         `last_login` datetime DEFAULT NULL,
                         PRIMARY KEY (`id`),
                         UNIQUE KEY `username` (`username`),
                         UNIQUE KEY `phone` (`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 创建触发器（使用 DELIMITER 分隔）
DELIMITER //
CREATE TRIGGER set_users_created_at
    BEFORE INSERT ON `users`
    FOR EACH ROW
BEGIN
    IF NEW.created_at IS NULL THEN
        SET NEW.created_at = NOW(3);
    END IF;
END //
DELIMITER ;

-- 用户邮箱绑定表（已添加status字段）
CREATE TABLE `user_mail_accounts` (
                                      `id` bigint(20) NOT NULL AUTO_INCREMENT,
                                      `user_id` bigint(20) NOT NULL,
                                      `email` varchar(100) NOT NULL,
                                      `auth_code` varchar(255) NOT NULL,
                                      `display_name` varchar(100) DEFAULT NULL,
                                      `status` enum('active', 'disabled') NOT NULL DEFAULT 'active',
                                      `created_at` datetime(3) NULL DEFAULT NULL,
                                      `updated_at` datetime(3) NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(3),
                                      PRIMARY KEY (`id`),
                                      FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
                                      UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 为user_mail_accounts表添加创建时间触发器
DELIMITER //
CREATE TRIGGER set_mail_accounts_created_at
    BEFORE INSERT ON `user_mail_accounts`
    FOR EACH ROW
BEGIN
    IF NEW.created_at IS NULL THEN
        SET NEW.created_at = NOW(3);
    END IF;
END //
DELIMITER ;

-- 邮件发送记录表
CREATE TABLE `email_records` (
                                 `id` bigint(20) NOT NULL AUTO_INCREMENT,
                                 `from_user_id` bigint(20) NOT NULL,
                                 `from_email` varchar(100) NOT NULL,
                                 `to_user_id` bigint(20) DEFAULT NULL,
                                 `to_email` varchar(100) NOT NULL,
                                 `status` enum('pending', 'success', 'fail') NOT NULL DEFAULT 'pending',
                                 `sent_at` datetime DEFAULT NULL,
                                 `recipient_type` ENUM('to', 'cc', 'bcc') NOT NULL DEFAULT 'to',
                                 `email_req_id` VARCHAR(36) NOT NULL,
                                 `retry_count` int NOT NULL DEFAULT 0,
                                 `last_checked_at` datetime DEFAULT NULL,
                                 PRIMARY KEY (`id`),
                                 FOREIGN KEY (`from_user_id`) REFERENCES `users` (`id`),
                                 INDEX `idx_user_id` (`from_user_id`),
                                 INDEX `idx_from_email` (`from_email`),
                                 INDEX `idx_to_email` (`to_email`),
                                 INDEX `idx_email_req_id` (`email_req_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 黑名单表
CREATE TABLE `blacklist` (
                             `id` bigint(20) NOT NULL AUTO_INCREMENT,
                             `email` varchar(100) NOT NULL,
                             `reason` varchar(255) DEFAULT NULL,
                             `created_by` bigint(20) NOT NULL,
                             `created_at` datetime(3) NULL DEFAULT NULL,
                             PRIMARY KEY (`id`),
                             UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 为blacklist表添加创建时间触发器
DELIMITER //
CREATE TRIGGER set_blacklist_created_at
    BEFORE INSERT ON `blacklist`
    FOR EACH ROW
BEGIN
    IF NEW.created_at IS NULL THEN
        SET NEW.created_at = NOW(3);
    END IF;
END //
DELIMITER ;