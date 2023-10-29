Command to run the service:
```bash
docker compose build && docker compose up
```

### Text of the task
To implement a service that will receive a full name (First Name, Last Name, Patronymic) via an API, enrich the response with the most likely age, gender, and nationality from open APIs, and store the data in a database. Upon request, provide information about the found individuals. The following needs to be implemented:

1. Set up REST methods
   1. For retrieving data with various filters and pagination.
   2. For deleting by identifier.
   3. For modifying an entity.
   4. For adding new people in the format:
      ```json
      {
        "name": "Dmitriy",
        "surname": "Ushakov",
        "patronymic": "Vasilevich" // optional
      }
      ```

2. Correct message enrich with
   1. Age - using https://api.agify.io/?name=Dmitriy
   2. Gender - using https://api.genderize.io/?name=Dmitriy
   3. Nationality - using https://api.nationalize.io/?name=Dmitriy

3. Store the enriched message in a PostgreSQL database (the database structure should be created through migrations).

4. Cover the code with debug and info logs.

5. Put the configuration data in a .env file.
