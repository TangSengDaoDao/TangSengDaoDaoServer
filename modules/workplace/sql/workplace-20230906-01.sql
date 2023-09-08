
-- +migrate Up


-- 工作台分类下app
create table `workplace_category_app`(
    id              bigint         not null primary key AUTO_INCREMENT,
    category_no     VARCHAR(40)    not null DEFAULT '',                -- 分类编号
    app_id          VARCHAR(40)    not null DEFAULT '',                -- appid
    sort_num        integer        not null DEFAULT 0,                 -- 排序编号
    created_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE unique INDEX workplace_category_app_cno_aid on `workplace_category_app` (category_no,app_id);
