/*
Navicat MySQL Data Transfer

Source Server         : localhost
Source Server Version : 50724
Source Host           : localhost:3306
Source Database       : mysql

Target Server Type    : MYSQL
Target Server Version : 50724
File Encoding         : 65001

Date: 2021-04-23 18:03:35
*/

SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for easy_chat_client
-- ----------------------------
DROP TABLE IF EXISTS `easy_chat_client`;
CREATE TABLE IF NOT EXISTS `easy_chat_client` (
  `id` bigint(10) NOT NULL AUTO_INCREMENT,
  `version` varchar(255) CHARACTER SET utf8 DEFAULT NULL,
  `version_number` int(4) DEFAULT NULL,
  `file_name` varchar(255) CHARACTER SET utf8 DEFAULT NULL,
  `file_size` bigint(12) DEFAULT NULL,
  `md5` varchar(255) CHARACTER SET utf8 DEFAULT NULL,
  `upload_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `upload_by` varchar(255) CHARACTER SET utf8 DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;
