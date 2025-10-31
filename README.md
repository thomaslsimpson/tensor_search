
The purpose of this project is to return a list of matching domain names given a set of keywords.

The domain names are stored in an sqlite database (rc_domain_embeds.sqlite3) along with text embeddings.

Other services should be able to use this module by providing a path to a properly formatted sqlite database and then calling:

get_matching_domains(keywords, country)

... which will return json in the format:

{
  "kw": keywords # the original string of keywords that we used to perform the search
  "cn": country, # the country that they sent us to start with (or "us" which will be the default)
  "dn": [],      # the list of strings (domains) that we returned which might be empty
  "err": 0       # an error code which is 0 on success or a error code number otherwise
}

So, if the user wrote something like:


domains := get_matching_domains("nice new trucks", "us") 

... they might get back:

{
  "kw": "nice new trucks",
  "cn": "us",
  "dn": ["ford.com", "chevy.com"],
  "err": 0
}


# Step 1: set up the project and write main() as "Hello WOrld" (test_go)

First, we need to set up teh basic go environment and get a Hello World Program going. 
This will ensure that our environment is set up properly and that we are not missing 
anything that we need to keep working.


# Step 2: make sure we can load the sqlite database and use it (test_sqlite)

Next, we will need to load the sqlite database from the local examples (rc_domain_embeds.sqlite3) and run a query to make sure it works.

This command should work locally before we start:

sqlite3 ./reference/rc_domain_embeds.sqlite3 ".schema"

... and display the table structure.


In order to know that our go code will be able to do what we need, we will write a test 
(that we can run using the go test system) that will run a query so we can verify that 
all is well:

This query should return 10 matches including ford.com:

    SELECT d2.domain, d2.country, d2.distance
    FROM domains AS d2
    WHERE d2.embedding MATCH (SELECT embedding FROM domains WHERE domain = 'ford.com')
    AND k = 10
    ORDER BY d2.distance


# Step 3: connect to Ollama and run a text encoding (test_ollama)

We will not use external libraries to do this because we do not need them. We only need to call 
Ollama via HTTP POST. The user will provide the URL and MODEL_NAME and we will run the HTTP POST
that sends JSON and expects a JSON result like this:

encode(text, ollama_url, model_name)

The POST URL will look like:

"{ollama_url}/api/embed"

... and the data payload sent will look like:

{
  "model": model_name,
  "input": text
}

... and the expected result will look like:

{
  "embeddings: [{embedding_1}, {embedding_2}, ... {embedding_N}]
}

... where the value of embedding_1 - N will be a tensor - a vector of floating point number that 
we will use to query the sqlite database for a match.

But in this step, we just want to ensure that we can call the Ollama server and get a response of 
the proper length adn format that we can read and store internally.

We will do this in the form of another go test that will verify that this is working.

# Step 4: perform a complete run (test_search)

We will write the actual function:

get_matching_domains(keywords, country)

... which will take the string of keywords, call the encode function from Step 3, then use 
the generated embedding to run a query to find matches in the database.

Running the keywors search for "nwe truck" should return these three top matches:

{
  'kw': 'new truck',
  'cn': 'us',
  'dn': ['napaonline.com', 'ford.com', 'suncentauto.com'],
  'err': 0
}


Step 5: text timing (test_time)

This test will run several keyword searches and check the time to run the Ollama check and to run the 
query to find the matches.

It will use these 10 search phrases:

"new truck"
"children's toys"
"adult toys"
"best socks"
"designer shoes"
"designer clothes"
"thrift clothes"
"hiking gear"
"automotive parts"
"shoe cleaning"

... and it will show the results for each test along with the time (in millisecsons) and the average time overall.






