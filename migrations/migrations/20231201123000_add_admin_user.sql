CREATE TABLE IF NOT EXISTS users (
                                     id TEXT PRIMARY KEY,
                                     username TEXT UNIQUE NOT NULL,
                                     password TEXT NOT NULL,
                                     email TEXT UNIQUE,
                                     role TEXT DEFAULT 'user',
                                     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Adiciona um usuário admin com senha "admin123"
-- A senha está hashada com bcrypt
-- Algoritmo: bcrypt
-- Custo: 10
-- Salt: $2a$10$V0z/TuZPkGjXdcNEThQYXu
-- Hash: $2a$10$V0z/TuZPkGjXdcNEThQYXu6MK6yH7GZ.8r6IHzQGKHpN8O9GnBUCG
-- Senha: admin123

INSERT INTO users (id, username, password, email, role)
VALUES (
           'admin',
           'admin',
           '$2a$10$V0z/TuZPkGjXdcNEThQYXu6MK6yH7GZ.8r6IHzQGKHpN8O9GnBUCG',
           'admin@example.com',
           'admin'
       )
    ON CONFLICT (username) DO NOTHING;
