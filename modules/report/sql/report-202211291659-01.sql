-- +migrate Up

ALTER TABLE `report_category` ADD COLUMN category_ename VARCHAR(100) not null DEFAULT '' comment '英文类别名称';


update  report_category set category_ename='Posting inappropriate content is harassing me' where category_no='10000';
update  report_category set category_ename='Fraudulent deception' where category_no='20000';
update  report_category set category_ename='This account may be compromised' where category_no='30000';


update  report_category set category_ename='Pornography' where category_no='10001';
update  report_category set category_ename='Illegal and contraband' where category_no='10002';
update  report_category set category_ename='Gamble' where category_no='10003';
update  report_category set category_ename='Political rumors' where category_no='10004';
update  report_category set category_ename='Violent and bloody' where category_no='10005';
update  report_category set category_ename='Other violations' where category_no='10006';

update  report_category set category_ename='Money received but not shipped' where category_no='20001';
update  report_category set category_ename='Financial loan scam money' where category_no='20002';
update  report_category set category_ename='Online part-time job scam money' where category_no='20003';
update  report_category set category_ename='Impersonation scam money' where category_no='20004';
update  report_category set category_ename='Send scam money for free' where category_no='20005';
update  report_category set category_ename='Other fraudulent deception' where category_no='20006';

