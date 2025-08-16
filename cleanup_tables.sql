-- このスクリプトは各期間のDenchokun.dbから不要なテーブルを削除します
-- 実行前にバックアップを取ることをお勧めします

-- DealPartnersテーブルを削除（System.dbに移行済み）
DROP TABLE IF EXISTS DealPartners;

-- Systemテーブルを削除（System.dbに移行済み）
DROP TABLE IF EXISTS System;

-- 残るテーブルを確認
SELECT name FROM sqlite_master WHERE type='table';