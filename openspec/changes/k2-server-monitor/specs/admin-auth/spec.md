## ADDED Requirements

### Requirement: Server generates admin credentials on first start
When the database is empty (no `admin_config` row), the server SHALL generate admin credentials on startup.

#### Scenario: First start generates credentials
- **WHEN** server starts and `admin_config` table is empty
- **THEN** system generates a unique 32+ hex char path hash
- **THEN** system generates a username from 2-3 random English words joined by hyphens
- **THEN** system generates a 16-character password containing uppercase, lowercase, digits, and special characters
- **THEN** system inserts a row into `admin_config` with the path hash
- **THEN** system inserts a row into `admin_user` with the username and plaintext password
- **THEN** system prints admin URL, username, and password to stdout

#### Scenario: Subsequent starts skip generation
- **WHEN** server starts and `admin_config` table already has a row
- **THEN** system uses existing credentials

### Requirement: Admin panel is accessible at /admin/<path_hash>/
The admin panel SHALL be served at a URL with the generated path hash.

#### Scenario: Access with correct hash
- **WHEN** user navigates to `/admin/<correct_hash>/`
- **THEN** system shows login page
- **WHEN** user navigates to `/admin/<correct_hash>/dashboard`
- **THEN** system requires authentication or shows login

#### Scenario: Access with incorrect hash returns 404
- **WHEN** user navigates to `/admin/<wrong_hash>/`
- **THEN** system returns 404

### Requirement: Login with username and password
The admin login page SHALL accept username and password.

#### Scenario: Successful login
- **WHEN** user submits correct username and password
- **THEN** system creates authenticated session
- **THEN** system redirects to admin dashboard
- **THEN** system resets `login_attempts` to 0 for that user

#### Scenario: Failed login increments attempts
- **WHEN** user submits incorrect password
- **THEN** system increments `login_attempts` by 1
- **THEN** system shows error message

### Requirement: Brute force protection
After 3 failed login attempts, the user SHALL be locked out for 1 minute.

#### Scenario: Lockout after 3 attempts
- **WHEN** `login_attempts` reaches 3
- **THEN** system sets `locked_until` to current time + 1 minute
- **THEN** system returns 429 Too Many Requests on further attempts
- **THEN** system shows lockout message with remaining time

#### Scenario: Lockout expires
- **WHEN** user tries to login after `locked_until` has passed
- **THEN** system allows login attempt again

### Requirement: Admin logout
The admin panel SHALL have a logout button.

#### Scenario: Logout
- **WHEN** authenticated user clicks logout
- **THEN** system clears session
- **THEN** system redirects to login page
### Migration: admin tables MUST be written to `migrations/20260724193038_create_tables.sql`

The `admin_config` and `admin_user` tables SHALL be created in the existing migration file `migrations/20260724193038_create_tables.sql` (Up section). The old `auth_user` table SHALL be dropped in the same migration.

### Requirement: k2 credentials displays admin info

The `k2 credentials` subcommand SHALL display admin URL, username, and password.

#### Scenario: Credentials displayed
- **WHEN** user runs `k2 credentials`
- **THEN** system reads from `admin_config` and `admin_user` tables
- **THEN** system prints:
  Admin URL: /admin/<hash>/
  Username: <username>
  Password: <password>
