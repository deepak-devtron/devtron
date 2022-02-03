ALTER TABLE chart_group DROP COLUMN IF EXISTS active;

ALTER TABLE chart_repo DROP COLUMN IF EXISTS deleted;

ALTER TABLE slack_config DROP COLUMN IF EXISTS deleted;

ALTER TABLE ses_config DROP COLUMN IF EXISTS deleted;

ALTER TABLE team ADD CONSTRAINT team_name_key UNIQUE (name);