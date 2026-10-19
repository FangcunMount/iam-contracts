-- Protocol selection must not disappear behind an active writer or Checker.
-- Recovery uses the reviewed compatible image and explicit protocol cutover.
SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'migration 000043 protocol gate is retained; use reviewed compatibility recovery';
