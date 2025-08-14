# streamshort


### Todo

### Authentication:

- POST /auth/signup (phone/email)

- POST /auth/login

- POST /auth/refresh

- POST /auth/logout

### User:

- GET /me

- GET /users/{id}/subscriptions

### Creator:

- POST /creators/onboard (KYC fields)

- GET /creators/{id}/dashboard (analytics)

- POST /creators/{id}/payout-request

### Content:

- POST /content/series (create)

- POST /content/series/{id}/episodes (create metadata)

- GET /content/series (list, filters: language, category)

- GET /content/series/{id}

- GET /content/episodes/{id}/manifest (returns signed HLS URL if authorized)

- POST /content/upload-url (generate signed URL for upload)


### Payment:

- POST /payments/create-subscription

- POST /payments/webhook (Razorpay callbacks)

- GET /payments/{id}/status

### Interaction:

- POST /episodes/{id}/like

-POST /episodes/{id}/rating

- GET /episodes/{id}/comments

- POST /episodes/{id}/comments

### Admin:

- GET /admin/uploads/pending

- POST /admin/approve-content
