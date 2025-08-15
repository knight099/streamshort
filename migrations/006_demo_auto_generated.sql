-- Migration: 006_demo_auto_generated.sql
-- Description: Demonstrate automated migration generation
-- Created: 2025-08-16
-- Auto-generated from Go models

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id uuid;default:gen_random_uuid() PRIMARY KEY DEFAULT gen_random_uuid() DEFAULT gen_random_uuid()