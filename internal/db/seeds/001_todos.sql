INSERT INTO users (name, email) VALUES
    ('Alice Johnson', 'alice@example.com'),
    ('Bob Smith',     'bob@example.com'),
    ('Carol White',   'carol@example.com');

INSERT INTO todos (title, completed, user_id) VALUES
    ('Buy groceries',        false, (SELECT id FROM users WHERE email = 'alice@example.com')),
    ('Write unit tests',     false, (SELECT id FROM users WHERE email = 'alice@example.com')),
    ('Read Clean Code',      true,  (SELECT id FROM users WHERE email = 'bob@example.com')),
    ('Set up CI pipeline',   false, (SELECT id FROM users WHERE email = 'bob@example.com')),
    ('Review pull requests', true,  (SELECT id FROM users WHERE email = 'carol@example.com')),
    ('Deploy to staging',    false, (SELECT id FROM users WHERE email = 'carol@example.com')),
    ('Fix login bug',        true,  (SELECT id FROM users WHERE email = 'alice@example.com'));
