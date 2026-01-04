UPDATE `users` SET `phone` = '' WHERE `phone` IS NULL;
ALTER TABLE `users`
    MODIFY COLUMN `phone` VARCHAR(20) NOT NULL COMMENT '手机号';
