-- +migrate Up

-- 组织
create table `organization`(
    id                      integer                not null primary key AUTO_INCREMENT,
    org_id                  VARCHAR(40)            not null default '', 
    short_no                VARCHAR(40)            not null default '',
    name                    VARCHAR(40)            not null default '', 
    creator                 VARCHAR(40)            not null default '',
    is_upload_logo          smallint               not null default 0,
    created_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP,
    updated_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX org_idx on `organization` (org_id);
CREATE UNIQUE INDEX short_nox on `organization` (short_no);
-- 组织员工
create table `organization_employee`(
    id                      integer                not null primary key AUTO_INCREMENT,
    org_id                  VARCHAR(40)            not null default '', 
    employee_uid            VARCHAR(40)            not null default '',
    employee_name           VARCHAR(100)           not null default '', 
    role                    smallint               not null DEFAULT 0,
    status                  smallint               not null default 0,
    employment_time         int                    not null default 0,                     -- 入职时间
    created_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP,
    updated_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX organization_employee_idx on `organization_employee` (org_id);
CREATE UNIQUE INDEX organization_employee_uidx on `organization_employee` (org_id,employee_uid);
-- 部门
create table `department`(
    id                      integer                not null primary key AUTO_INCREMENT,
    org_id                  VARCHAR(40)            not null default '',
    dept_id                 VARCHAR(40)            not null default '', 
    name                    VARCHAR(100)           not null default '',
    parent_id               VARCHAR(40)            not null default '',
    short_no                VARCHAR(40)            not null default '',
    is_created_group        smallint               not null default 0,
    created_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP,
    updated_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX department_idx on `department` (dept_id);
-- 部门员工
create table `department_employee`(
    id                      integer                not null primary key AUTO_INCREMENT,
    org_id                  VARCHAR(40)            not null default '',
    dept_id                 VARCHAR(40)            not null default '', 
    employee_uid            VARCHAR(40)            not null default '', 
    employee_id             VARCHAR(40)            not null default '',                     -- 工号
    workforce_type          VARCHAR(40)            not null default '',                     -- 人员类型
    job_title               VARCHAR(40)            not null default '',                     -- 职务
    created_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP,
    updated_at              timeStamp              not null DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX department_employee_uidx on `department_employee` (dept_id,employee_uid);
CREATE INDEX department_employee_org_id_dept_idx on `department_employee` (dept_id,org_id);
