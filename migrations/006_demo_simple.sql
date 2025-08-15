-- Migration: 006_demo_simple.sql
-- Description: Demonstrate automated migration generation
-- Created: 2025-08-16
-- Auto-generated from Go models

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id uuid;default:gen_random_uuid() PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(255) NOT NULL UNIQUE,
    createdat TIMESTAMP WITH TIME ZONE,
    updatedat TIMESTAMP WITH TIME ZONE,
    deletedat TEXT,
    creatorprofile TEXT,

);


-- Create otptransactions table
CREATE TABLE IF NOT EXISTS otptransactions (
    id uuid;default:gen_random_uuid() PRIMARY KEY DEFAULT gen_random_uuid(),
    txnid VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(255) NOT NULL,
    otp VARCHAR(255) NOT NULL,
    expiresat TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN DEFAULT false,
    createdat TIMESTAMP WITH TIME ZONE,
    updatedat TIMESTAMP WITH TIME ZONE,
    deletedat TEXT,

);


-- Create refreshtokens table
CREATE TABLE IF NOT EXISTS refreshtokens (
    id uuid;default:gen_random_uuid() PRIMARY KEY DEFAULT gen_random_uuid(),
    token VARCHAR(255) NOT NULL UNIQUE,
    userid uuid NOT NULL,
    expiresat TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN DEFAULT false,
    createdat TIMESTAMP WITH TIME ZONE,
    updatedat TIMESTAMP WITH TIME ZONE,
    deletedat TEXT,

);


-- Create creatorprofiles table
CREATE TABLE IF NOT EXISTS creatorprofiles (
    id uuid;default:gen_random_uuid() PRIMARY KEY DEFAULT gen_random_uuid(),
    userid uuid;not NOT NULL UNIQUE,
    displayname VARCHAR(255) NOT NULL,
    bio VARCHAR(255),
    kyc_document_s3_path VARCHAR(255),
    kycstatus VARCHAR(255) DEFAULT 'pending';check:kyc_status,
    payoutdetails TEXT,
    rating decimal(3,2),
    createdat TIMESTAMP WITH TIME ZONE,
    updatedat TIMESTAMP WITH TIME ZONE,
    deletedat TEXT,
    user TEXT,

);


-- Create payoutdetailss table
CREATE TABLE IF NOT EXISTS payoutdetailss (
    id uuid;default:gen_random_uuid() PRIMARY KEY DEFAULT gen_random_uuid(),
    creatorid uuid;not NOT NULL UNIQUE,
    bankname VARCHAR(255),
    accountnumber VARCHAR(255),
    ifsccode VARCHAR(255),
    accountholder VARCHAR(255),
    createdat TIMESTAMP WITH TIME ZONE,
    updatedat TIMESTAMP WITH TIME ZONE,
    deletedat TEXT,

);


-- Create creatoranalyticss table
CREATE TABLE IF NOT EXISTS creatoranalyticss (
    id uuid;default:gen_random_uuid() PRIMARY KEY DEFAULT gen_random_uuid(),
    creatorid uuid;not NOT NULL,
    date date;not NOT NULL,
    views BIGINT DEFAULT 0,
    watchtimeseconds BIGINT DEFAULT 0,
    earnings decimal(10,2);default:0 DEFAULT 0,
    createdat TIMESTAMP WITH TIME ZONE,
    updatedat TIMESTAMP WITH TIME ZONE,
    deletedat TEXT,
    creator TEXT,

);



-- Migration completed successfully
