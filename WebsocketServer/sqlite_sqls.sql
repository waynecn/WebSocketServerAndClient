alter table chat_upload_files add user_id INTEGER after upload_user;
alter table chat_upload_files add to_user_id INTEGER after user_id;
alter table chat_user add user_type INTEGER after `password` default 2;

//可以显示chat_user的建表语句
.schema chat_user 
