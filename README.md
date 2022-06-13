# users-go
This is an internal service which is used to save the user related information. 
This service saves user information and user access information . The service saves the doornumber as a map so scaling more doors is easy with no change in structure.

# Endpoints 
GET /getuser - This endpoint takes username as query param and returns all the userinformation. 

POST /updateuseraccess - This endpoint is used to update the access for a user if the user is admin. Only Admin users can update the access.

POST /authenticate - This endpoint checks if the user has access to the door the user is trying to access.

# Database
Dbname - users

# Database Instance Name 
db-users- <env>

  env can be test,perf or prod
