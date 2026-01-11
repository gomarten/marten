# JWT Auth Example

JWT authentication with protected routes and role-based access.

## Run

```bash
go run .
```

## Endpoints

- `POST /auth/login` - Login (user@example.com / password123)
- `POST /auth/register` - Register new user
- `GET /api/me` - Current user (requires token)
- `GET /api/profile` - User profile (requires token)
- `PUT /api/profile` - Update profile (requires token)
- `GET /admin/users` - Admin only (admin@example.com)

## Usage

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}' | jq -r .token)

# Access protected route
curl http://localhost:3000/api/me -H "Authorization: Bearer $TOKEN"
```
