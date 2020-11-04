/*
Navicat MySQL Data Transfer

Source Server         : localhost
Source Server Version : 50724
Source Host           : localhost:3306
Source Database       : mysql

Target Server Type    : MYSQL
Target Server Version : 50724
File Encoding         : 65001

Date: 2020-06-24 00:39:08
*/

SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for chat_user
-- ----------------------------
DROP TABLE IF EXISTS `chat_upload_files`;
CREATE TABLE `chat_upload_files` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `file_name` varchar(255) CHARACTER SET utf8 DEFAULT NULL,
  `create_time` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=latin1;
