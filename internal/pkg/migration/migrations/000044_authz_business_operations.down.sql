-- Operation IDs, frozen plans and typed results remain usable after recovery.
SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'migration 000044 business operation evidence is retained; use reviewed compatibility recovery';
