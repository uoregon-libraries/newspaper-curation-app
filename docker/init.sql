-- MySQL dump 10.19  Distrib 10.3.39-MariaDB, for debian-linux-gnu (x86_64)
--
-- Host: 127.0.0.1    Database: nca
-- ------------------------------------------------------
-- Server version	11.6.2-MariaDB-ubu2404

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Current Database: `nca`
--

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `nca` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_uca1400_ai_ci */;

USE `nca`;

--
-- Table structure for table `actions`
--

DROP TABLE IF EXISTS `actions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `actions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `object_type` tinytext DEFAULT NULL,
  `object_id` int(11) NOT NULL,
  `action_type` tinytext DEFAULT NULL,
  `user_id` int(11) NOT NULL,
  `message` text DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `actions_created_at` (`created_at`),
  KEY `actions_object_id` (`object_id`),
  KEY `actions_action_type` (`action_type`(255)),
  KEY `actions_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `actions`
--

LOCK TABLES `actions` WRITE;
/*!40000 ALTER TABLE `actions` DISABLE KEYS */;
/*!40000 ALTER TABLE `actions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `audit_logs`
--

DROP TABLE IF EXISTS `audit_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `audit_logs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `when` datetime DEFAULT NULL,
  `ip` tinytext DEFAULT NULL,
  `user` tinytext DEFAULT NULL,
  `action` tinytext DEFAULT NULL,
  `message` mediumtext DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `audit_logs_when` (`when`),
  KEY `audit_logs_user` (`user`(255)),
  KEY `audit_logs_action` (`action`(255))
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `audit_logs`
--

LOCK TABLES `audit_logs` WRITE;
/*!40000 ALTER TABLE `audit_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `audit_logs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `batches`
--

DROP TABLE IF EXISTS `batches`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `batches` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `marc_org_code` tinytext NOT NULL,
  `created_at` datetime DEFAULT NULL,
  `name` tinytext NOT NULL,
  `status` tinytext NOT NULL,
  `location` tinytext NOT NULL,
  `went_live_at` datetime DEFAULT NULL,
  `archived_at` datetime DEFAULT NULL,
  `need_staging_purge` tinyint(4) DEFAULT NULL,
  `oni_agent_job_id` bigint(20) DEFAULT NULL,
  `full_name` tinytext DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `batches_marc_org_code` (`marc_org_code`(255)),
  KEY `batches_created_at` (`created_at`),
  KEY `batches_status` (`status`(255)),
  KEY `batches_went_live_at` (`went_live_at`),
  KEY `batches_archived_at` (`archived_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `batches`
--

LOCK TABLES `batches` WRITE;
/*!40000 ALTER TABLE `batches` DISABLE KEYS */;
/*!40000 ALTER TABLE `batches` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `batches_flagged_issues`
--

DROP TABLE IF EXISTS `batches_flagged_issues`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `batches_flagged_issues` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `flagged_by_user_id` int(11) NOT NULL,
  `batch_id` int(11) NOT NULL,
  `issue_id` int(11) NOT NULL,
  `reason` text NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `bfi_batch_issue` (`batch_id`,`issue_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `batches_flagged_issues`
--

LOCK TABLES `batches_flagged_issues` WRITE;
/*!40000 ALTER TABLE `batches_flagged_issues` DISABLE KEYS */;
/*!40000 ALTER TABLE `batches_flagged_issues` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb4 */ ;
/*!50003 SET character_set_results = utf8mb4 */ ;
/*!50003 SET collation_connection  = utf8mb4_uca1400_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`nca`@`%`*/ /*!50003 TRIGGER `batches_flagged_issues_created_at`
  BEFORE INSERT ON `batches_flagged_issues`
  FOR EACH ROW
  SET NEW.created_at = NOW() */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `goose_db_version`
--

DROP TABLE IF EXISTS `goose_db_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `goose_db_version` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `version_id` bigint(20) NOT NULL,
  `is_applied` tinyint(1) NOT NULL,
  `tstamp` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=68 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `goose_db_version`
--

LOCK TABLES `goose_db_version` WRITE;
/*!40000 ALTER TABLE `goose_db_version` DISABLE KEYS */;
INSERT INTO `goose_db_version` VALUES (1,0,1,'2024-12-26 23:09:18'),(2,20160219083459,1,'2024-12-26 23:09:18'),(3,20160222091642,1,'2024-12-26 23:09:18'),(4,20160226181704,1,'2024-12-26 23:09:18'),(5,20160301084706,1,'2024-12-26 23:09:19'),(6,20160302115537,1,'2024-12-26 23:09:19'),(7,20160517083435,1,'2024-12-26 23:09:19'),(8,20160606125420,1,'2024-12-26 23:09:19'),(9,20160627130506,1,'2024-12-26 23:09:19'),(10,20170901100204,1,'2024-12-26 23:09:19'),(11,20170911133800,1,'2024-12-26 23:09:19'),(12,20170918153100,1,'2024-12-26 23:09:19'),(13,20170926091700,1,'2024-12-26 23:09:19'),(14,20171019155400,1,'2024-12-26 23:09:19'),(15,20171027104100,1,'2024-12-26 23:09:19'),(16,20171027162100,1,'2024-12-26 23:09:19'),(17,20171030125000,1,'2024-12-26 23:09:19'),(18,20171207150000,1,'2024-12-26 23:09:19'),(19,20171219102000,1,'2024-12-26 23:09:19'),(20,20180129180000,1,'2024-12-26 23:09:19'),(21,20180205160600,1,'2024-12-26 23:09:19'),(22,20180207141300,1,'2024-12-26 23:09:19'),(23,20180213145800,1,'2024-12-26 23:09:19'),(24,20180213150600,1,'2024-12-26 23:09:19'),(25,20180215102900,1,'2024-12-26 23:09:19'),(26,20180306093900,1,'2024-12-26 23:09:19'),(27,20180309132300,1,'2024-12-26 23:09:20'),(28,20180410100235,1,'2024-12-26 23:09:20'),(29,20180418140400,1,'2024-12-26 23:09:20'),(30,20180418150700,1,'2024-12-26 23:09:20'),(31,20180426125800,1,'2024-12-26 23:09:20'),(32,20180608093900,1,'2024-12-26 23:09:20'),(33,20180904131800,1,'2024-12-26 23:09:20'),(34,20180904151800,1,'2024-12-26 23:09:20'),(35,20181119121200,1,'2024-12-26 23:09:20'),(36,20190415075218,1,'2024-12-26 23:09:20'),(37,20190808114000,1,'2024-12-26 23:09:20'),(38,20190917153600,1,'2024-12-26 23:09:20'),(39,20190918082705,1,'2024-12-26 23:09:20'),(40,20191230063200,1,'2024-12-26 23:09:20'),(41,20191231145200,1,'2024-12-26 23:09:20'),(42,20200228143700,1,'2024-12-26 23:09:20'),(43,20200427163800,1,'2024-12-26 23:09:20'),(44,20200430092900,1,'2024-12-26 23:09:20'),(45,20200501121800,1,'2024-12-26 23:09:20'),(46,20200501142600,1,'2024-12-26 23:09:20'),(47,20200505140000,1,'2024-12-26 23:09:20'),(48,20200728070500,1,'2024-12-26 23:09:20'),(49,20200910142500,1,'2024-12-26 23:09:20'),(50,20210208112800,1,'2024-12-26 23:09:22'),(51,20210921044200,1,'2024-12-26 23:09:22'),(52,20211013131100,1,'2024-12-26 23:09:22'),(53,20220330110400,1,'2024-12-26 23:09:22'),(54,20220723124500,1,'2024-12-26 23:09:22'),(55,20220907042700,1,'2024-12-26 23:09:22'),(56,20221130112700,1,'2024-12-26 23:09:22'),(57,20230626060501,1,'2024-12-26 23:09:23'),(58,20230724080000,1,'2024-12-26 23:09:23'),(59,20230731125500,1,'2024-12-26 23:09:23'),(60,20240731074000,1,'2024-12-26 23:09:23'),(61,20240805110000,1,'2024-12-26 23:09:23'),(62,20240821081100,1,'2024-12-26 23:09:23'),(63,20240926113000,1,'2024-12-26 23:09:24'),(64,20240930112500,1,'2024-12-26 23:09:24'),(65,20240930121200,1,'2024-12-26 23:09:24'),(66,20241220071500,1,'2024-12-26 23:09:24'),(67,20241220073000,1,'2024-12-26 23:09:24');
/*!40000 ALTER TABLE `goose_db_version` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `issues`
--

DROP TABLE IF EXISTS `issues`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `issues` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `marc_org_code` tinytext DEFAULT NULL,
  `lccn` tinytext DEFAULT NULL,
  `date` tinytext DEFAULT NULL,
  `date_as_labeled` tinytext DEFAULT NULL,
  `volume` tinytext DEFAULT NULL,
  `issue` tinytext DEFAULT NULL,
  `edition` tinyint(4) NOT NULL,
  `edition_label` tinytext DEFAULT NULL,
  `page_labels_csv` mediumtext DEFAULT NULL,
  `location` tinytext DEFAULT NULL,
  `is_from_scanner` tinyint(4) DEFAULT NULL,
  `metadata_entry_user_id` int(11) DEFAULT NULL,
  `reviewed_by_user_id` int(11) DEFAULT NULL,
  `workflow_owner_id` int(11) DEFAULT NULL,
  `workflow_owner_expires_at` datetime DEFAULT NULL,
  `workflow_step` tinytext DEFAULT NULL,
  `rejected_by_user_id` int(11) DEFAULT NULL,
  `human_name` tinytext DEFAULT NULL,
  `metadata_approved_at` datetime DEFAULT NULL,
  `backup_location` tinytext DEFAULT NULL,
  `batch_id` int(11) NOT NULL DEFAULT 0,
  `ignored` tinyint(4) NOT NULL DEFAULT 0,
  `draft_comment` text DEFAULT NULL,
  `metadata_entered_at` datetime DEFAULT NULL,
  `page_count` int(11) DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `issues_marc_org_code` (`marc_org_code`(255)),
  KEY `issues_lccn` (`lccn`(255)),
  KEY `issues_metadata_entry_user_id` (`metadata_entry_user_id`),
  KEY `issues_reviewed_by_user_id` (`reviewed_by_user_id`),
  KEY `issues_workflow_owner_id` (`workflow_owner_id`),
  KEY `issues_workflow_step` (`workflow_step`(255)),
  KEY `issues_rejected_by_user_id` (`rejected_by_user_id`),
  KEY `issues_metadata_approved_at` (`metadata_approved_at`),
  KEY `issues_batch_id` (`batch_id`),
  KEY `issue_metadata_entry` (`metadata_entered_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `issues`
--

LOCK TABLES `issues` WRITE;
/*!40000 ALTER TABLE `issues` DISABLE KEYS */;
/*!40000 ALTER TABLE `issues` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `job_logs`
--

DROP TABLE IF EXISTS `job_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `job_logs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `job_id` bigint(20) NOT NULL,
  `created_at` datetime DEFAULT NULL,
  `log_level` tinytext DEFAULT NULL,
  `message` mediumtext DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  KEY `job_logs_job_id` (`job_id`),
  KEY `job_logs_created_at` (`created_at`),
  KEY `job_logs_log_level` (`log_level`(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `job_logs`
--

LOCK TABLES `job_logs` WRITE;
/*!40000 ALTER TABLE `job_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `job_logs` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb4_uca1400_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`nca`@`%`*/ /*!50003 TRIGGER `job_logs_created_at`
  BEFORE INSERT ON `job_logs`
  FOR EACH ROW
  SET NEW.created_at = NOW() */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `jobs`
--

DROP TABLE IF EXISTS `jobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `jobs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `job_type` tinytext DEFAULT NULL,
  `object_id` bigint(20) NOT NULL,
  `status` tinytext DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `started_at` datetime DEFAULT NULL,
  `run_at` datetime DEFAULT NULL,
  `extra_data` mediumtext DEFAULT NULL,
  `object_type` tinytext NOT NULL,
  `retry_count` int(11) DEFAULT NULL,
  `pipeline_id` int(11) NOT NULL,
  `sequence` tinyint(4) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  KEY `jobs_created_at` (`created_at`),
  KEY `jobs_job_type` (`job_type`(255)),
  KEY `jobs_object_id` (`object_id`),
  KEY `jobs_status` (`status`(255)),
  KEY `jobs_pipeline_id` (`pipeline_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `jobs`
--

LOCK TABLES `jobs` WRITE;
/*!40000 ALTER TABLE `jobs` DISABLE KEYS */;
/*!40000 ALTER TABLE `jobs` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb3 */ ;
/*!50003 SET character_set_results = utf8mb3 */ ;
/*!50003 SET collation_connection  = utf8mb4_uca1400_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`nca`@`%`*/ /*!50003 TRIGGER `jobs_created_at`
  BEFORE INSERT ON `jobs`
  FOR EACH ROW
  SET NEW.created_at = UTC_TIMESTAMP() */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Temporary table structure for view `moc_issue_aggregation`
--

DROP TABLE IF EXISTS `moc_issue_aggregation`;
/*!50001 DROP VIEW IF EXISTS `moc_issue_aggregation`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8;
/*!50001 CREATE VIEW `moc_issue_aggregation` AS SELECT
 1 AS `id`,
  1 AS `code`,
  1 AS `name`,
  1 AS `workflow_step`,
  1 AS `issue_count`,
  1 AS `total_pages` */;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `mocs`
--

DROP TABLE IF EXISTS `mocs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `mocs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `code` tinytext DEFAULT NULL,
  `name` mediumtext DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `mocs`
--

LOCK TABLES `mocs` WRITE;
/*!40000 ALTER TABLE `mocs` DISABLE KEYS */;
INSERT INTO `mocs` VALUES (1,'oru','University of Oregon Libraries; Eugene, OR'),(2,'hoodriverlibrary','Hood River County Library District; Hood River, OR');
/*!40000 ALTER TABLE `mocs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `pipelines`
--

DROP TABLE IF EXISTS `pipelines`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pipelines` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `started_at` datetime DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `name` tinytext NOT NULL,
  `object_type` tinytext DEFAULT NULL,
  `object_id` int(11) DEFAULT NULL,
  `description` text NOT NULL,
  PRIMARY KEY (`id`),
  KEY `pipelines_name` (`name`(255)),
  KEY `pipelines_created_at` (`created_at`),
  KEY `pipelines_started_at` (`started_at`),
  KEY `pipelines_object_type` (`object_type`(255)),
  KEY `pipelines_object_id` (`object_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pipelines`
--

LOCK TABLES `pipelines` WRITE;
/*!40000 ALTER TABLE `pipelines` DISABLE KEYS */;
/*!40000 ALTER TABLE `pipelines` ENABLE KEYS */;
UNLOCK TABLES;
/*!50003 SET @saved_cs_client      = @@character_set_client */ ;
/*!50003 SET @saved_cs_results     = @@character_set_results */ ;
/*!50003 SET @saved_col_connection = @@collation_connection */ ;
/*!50003 SET character_set_client  = utf8mb4 */ ;
/*!50003 SET character_set_results = utf8mb4 */ ;
/*!50003 SET collation_connection  = utf8mb4_uca1400_ai_ci */ ;
/*!50003 SET @saved_sql_mode       = @@sql_mode */ ;
/*!50003 SET sql_mode              = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;
DELIMITER ;;
/*!50003 CREATE*/ /*!50017 DEFINER=`nca`@`%`*/ /*!50003 TRIGGER `pipelines_created_at`
  BEFORE INSERT ON `pipelines`
  FOR EACH ROW
  SET NEW.created_at = UTC_TIMESTAMP() */;;
DELIMITER ;
/*!50003 SET sql_mode              = @saved_sql_mode */ ;
/*!50003 SET character_set_client  = @saved_cs_client */ ;
/*!50003 SET character_set_results = @saved_cs_results */ ;
/*!50003 SET collation_connection  = @saved_col_connection */ ;

--
-- Table structure for table `titles`
--

DROP TABLE IF EXISTS `titles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `titles` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` tinytext DEFAULT NULL,
  `lccn` tinytext DEFAULT NULL,
  `rights` tinytext DEFAULT NULL,
  `valid_lccn` tinyint(4) DEFAULT NULL,
  `sftp_dir` tinytext DEFAULT NULL,
  `sftp_user` tinytext DEFAULT NULL,
  `sftp_pass` tinytext DEFAULT NULL,
  `marc_title` tinytext DEFAULT NULL,
  `marc_location` tinytext DEFAULT NULL,
  `is_historic` tinyint(4) NOT NULL DEFAULT 0,
  `embargo_period` tinytext DEFAULT '0',
  `lang_code3` tinytext DEFAULT NULL,
  `sftp_connected` tinyint(4) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `titles`
--

LOCK TABLES `titles` WRITE;
/*!40000 ALTER TABLE `titles` DISABLE KEYS */;
INSERT INTO `titles` VALUES (2,'Appeal tribune. (Silverton, Or.)','2004260523','',1,NULL,'2004260523','pass','Appeal tribune.','Silverton, Or.',0,'','eng',0),(3,'Just out. (Portland, OR)','2013202554','',1,NULL,'2013202554','pass','Just out.','Portland, OR',0,'','eng',0),(4,'Northwest labor press. (Portland , Ore.)','2018252080','',1,NULL,'2018252080','pass','Northwest labor press.','Portland , Ore.',0,'','eng',0),(5,'Siletz news / (Siletz, OR)','2021242619','',1,NULL,'2021242619','pass','Siletz news /','Siletz, OR',0,'','eng',0),(6,'Keizertimes. (Salem, Or.)','sn00063621','',1,NULL,'sn00063621','pass','Keizertimes.','Salem, Or.',0,'','eng',0),(7,'The daily Astorian. (Astoria, Or.)','sn83008376','',1,NULL,'sn83008376','pass','The daily Astorian.','Astoria, Or.',0,'','eng',0),(8,'East Oregonian : E.O. (Pendleton, OR)','sn88086023','',1,NULL,'sn88086023','pass','East Oregonian : E.O.','Pendleton, OR',0,'','eng',0),(9,'Wallowa County chieftain. (Enterprise, Wallowa County, Or.)','sn90057139','',1,NULL,'sn90057139','pass','Wallowa County chieftain.','Enterprise, Wallowa County, Or.',0,'','eng',0),(10,'Cottage Grove sentinel. (Cottage Grove, Or.)','sn96088073','',1,NULL,'sn96088073','pass','Cottage Grove sentinel.','Cottage Grove, Or.',0,'','eng',0),(11,'Polk County itemizer observer. (Dallas, Or)','sn96088087','',1,NULL,'sn96088087','pass','Polk County itemizer observer.','Dallas, Or',0,'','eng',0),(12,'Vernonia eagle. (Vernonia, Or.)','sn99063854','',1,NULL,'sn99063854','pass','Vernonia eagle.','Vernonia, Or.',0,'','eng',0);
/*!40000 ALTER TABLE `titles` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `login` tinytext DEFAULT NULL,
  `roles` text DEFAULT NULL,
  `deactivated` tinyint(4) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES (3,'admin','admin',0),(4,'titlemanager','title manager',0),(5,'issuecurator','issue curator',0),(6,'issuereviewer','issue reviewer',0),(7,'issuemanager','issue manager',0),(8,'usermanager','user manager',0),(9,'marcorgcodemanager','marc org code manager',0),(10,'workflowmanager','workflow manager',0),(11,'batchbuilder','batch builder',0),(12,'batchreviewer','batch reviewer',0),(13,'batchloader','batch loader',0);
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Dumping routines for database 'nca'
--

--
-- Current Database: `nca`
--

USE `nca`;

--
-- Final view structure for view `moc_issue_aggregation`
--

/*!50001 DROP VIEW IF EXISTS `moc_issue_aggregation`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_uca1400_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`nca`@`%` SQL SECURITY DEFINER */
/*!50001 VIEW `moc_issue_aggregation` AS select `m`.`id` AS `id`,`m`.`code` AS `code`,`m`.`name` AS `name`,`i`.`workflow_step` AS `workflow_step`,count(`i`.`id`) AS `issue_count`,sum(`i`.`page_count`) AS `total_pages` from (`mocs` `m` join `issues` `i` on(`i`.`marc_org_code` = `m`.`code`)) where `i`.`ignored` = 0 group by `i`.`marc_org_code`,`i`.`workflow_step` order by `i`.`marc_org_code`,`i`.`workflow_step` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2024-12-26 15:10:44
