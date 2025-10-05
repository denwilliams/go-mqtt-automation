-- Remove comparison strategies

DELETE FROM strategies WHERE id IN ('less_than', 'greater_than', 'less_than_or_equal', 'greater_than_or_equal');
