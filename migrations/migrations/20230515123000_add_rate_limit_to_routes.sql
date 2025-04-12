-- Adicionar coluna de limite de taxa à tabela de rotas
ALTER TABLE routes ADD COLUMN rate_limit_per_minute INT DEFAULT 0;
ALTER TABLE routes ADD COLUMN rate_limit_per_hour INT DEFAULT 0;
ALTER TABLE routes ADD COLUMN rate_limit_burst_factor REAL DEFAULT 1.5;

-- Adicionar índice para consultas de métricas
CREATE INDEX IF NOT EXISTS idx_routes_call_count ON routes(call_count);

-- Adicionar índice para filtragem por status ativo
CREATE INDEX IF NOT EXISTS idx_routes_is_active ON routes(is_active);