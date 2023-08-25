-- +migrate Up

--  gitee_user 用户信息
CREATE TABLE IF NOT EXISTS gitee_user(
    id BIGINT PRIMARY KEY DEFAULT 0 COMMENT '用户 ID',
    `login` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '用户名',
    name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '用户姓名',
    email VARCHAR(255) NOT NULL DEFAULT '' COMMENT '用户邮箱',
    bio VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户简介',
    avatar_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户头像 URL',
    blog VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户博客 URL',
    events_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户事件 URL',
    followers INT NOT NULL DEFAULT 0 COMMENT '用户粉丝数',
    followers_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户粉丝 URL',
    following INT NOT NULL DEFAULT 0 COMMENT '用户关注数',
    following_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户关注 URL',
    gists_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户 Gist URL',
    html_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户主页 URL',
    member_role VARCHAR(100) NOT NULL DEFAULT '' COMMENT '用户角色',
    organizations_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户组织 URL',
    public_gists INT NOT NULL DEFAULT 0 COMMENT '用户公开 Gist 数',
    public_repos INT NOT NULL DEFAULT 0 COMMENT '用户公开仓库数',
    received_events_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户接收事件 URL',
    remark VARCHAR(100) NOT NULL DEFAULT '' COMMENT '企业备注名',
    repos_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户仓库 URL',
    stared INT NOT NULL DEFAULT 0 COMMENT '用户收藏数',
    starred_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户收藏 URL',
    subscriptions_url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户订阅 URL',
    url VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户 URL',
    watched INT NOT NULL DEFAULT 0 COMMENT '用户关注的仓库数',
    weibo VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '用户微博 URL',
    `type` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '用户类型',
    `gitee_created_at` VARCHAR(30) NOT NULL DEFAULT '' COMMENT 'gitee用户创建时间',
    `gitee_updated_at` VARCHAR(30) NOT NULL DEFAULT '' COMMENT 'gitee用户更新时间',
    created_at timeStamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at timeStamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间'
);
CREATE unique INDEX gitee_user_login on `gitee_user` (`login`);

ALTER TABLE `user` ADD COLUMN gitee_uid VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'gitee的用户id'; 

-- gthub用户
CREATE TABLE IF NOT EXISTS github_user (
    id BIGINT PRIMARY KEY DEFAULT 0 COMMENT '用户 ID',
    login VARCHAR(100) NOT NULL COMMENT '登录名',
    node_id VARCHAR(255) NOT NULL COMMENT '节点ID',
    avatar_url VARCHAR(1000) NOT NULL COMMENT '头像URL',
    gravatar_id VARCHAR(1000) NOT NULL COMMENT 'Gravatar ID',
    url VARCHAR(1000) NOT NULL COMMENT 'GitHub URL',
    html_url VARCHAR(1000) NOT NULL COMMENT 'GitHub HTML URL',
    followers_url VARCHAR(1000) NOT NULL COMMENT '关注者URL',
    following_url VARCHAR(1000) NOT NULL COMMENT '被关注者URL',
    gists_url VARCHAR(1000) NOT NULL COMMENT '代码片段URL',
    starred_url VARCHAR(1000) NOT NULL COMMENT '收藏URL',
    subscriptions_url VARCHAR(1000) NOT NULL COMMENT '订阅URL',
    organizations_url VARCHAR(1000) NOT NULL COMMENT '组织URL',
    repos_url VARCHAR(1000) NOT NULL COMMENT '仓库URL',
    events_url VARCHAR(1000) NOT NULL COMMENT '事件URL',
    received_events_url VARCHAR(1000) NOT NULL COMMENT '接收事件URL',
    `type` VARCHAR(100) NOT NULL COMMENT '用户类型',
    site_admin BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否为管理员',
    name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '姓名',
    company VARCHAR(100) NOT NULL DEFAULT '' COMMENT '公司',
    blog VARCHAR(100) NOT NULL DEFAULT '' COMMENT '博客',
    location VARCHAR(255) NOT NULL DEFAULT '' COMMENT '所在地',
    email VARCHAR(100) NOT NULL DEFAULT '' COMMENT '电子邮件',
    hireable BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否可被雇佣',
    bio VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '个人简介',
    twitter_username VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'Twitter 用户名',
    public_repos INT NOT NULL DEFAULT 0 COMMENT '公共仓库数量',
    public_gists INT NOT NULL DEFAULT 0 COMMENT '公共代码片段数量',
    followers INT NOT NULL DEFAULT 0 COMMENT '关注者数量',
    following INT NOT NULL DEFAULT 0 COMMENT '被关注者数量',
    github_created_at VARCHAR(30)  NOT NULL DEFAULT '' COMMENT '创建时间',
    github_updated_at VARCHAR(30)  NOT NULL DEFAULT '' COMMENT '更新时间',
    private_gists INT NOT NULL DEFAULT 0 COMMENT '私有代码片段数量',
    total_private_repos INT NOT NULL DEFAULT 0 COMMENT '私有仓库总数',
    owned_private_repos INT NOT NULL DEFAULT 0 COMMENT '拥有的私有仓库数量',
    disk_usage INT NOT NULL DEFAULT 0 COMMENT '磁盘使用量',
    collaborators INT NOT NULL DEFAULT 0 COMMENT '协作者数量',
    two_factor_authentication BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否启用两步验证',
    created_at timeStamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at timeStamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间'
);

CREATE unique INDEX github_user_login on `github_user` (`login`);

ALTER TABLE `user` ADD COLUMN github_uid VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'github的用户id'; 

