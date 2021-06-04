/*
Navicat MySQL Data Transfer

Source Server         : localhost
Source Server Version : 50724
Source Host           : localhost:3306
Source Database       : mysql

Target Server Type    : MYSQL
Target Server Version : 50724
File Encoding         : 65001

Date: 2020-11-13 12:05:26
*/

SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for chat_upload_files
-- ----------------------------
DROP TABLE IF EXISTS `chat_upload_files`;
CREATE TABLE IF NOT EXISTS `chat_upload_files` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `file_name` varchar(255) CHARACTER SET utf8 DEFAULT NULL,
  `file_size` bigint(20) DEFAULT NULL COMMENT '文件大小',
  `upload_user` varchar(255) CHARACTER SET utf8 DEFAULT NULL COMMENT '上传者',
  `create_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8;
