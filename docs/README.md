# Public Decision Tree
The *Public Decision Tree* is a simple little social sandbox style website that lets users answer yes/no questions. When a user reaches the end of a particular line of questioning, they are prompted to create their own and extend that branch. The purpose of the website is nothing other than to be a small fun exploratory artifact of online interactions.

The project is built in Go, with SQLite as a database, Go templates, Bootstrap CSS, and vanilla JS used for dynamic page updating while the tree is being traversed.
## Setup
>[!WARNING]
>GO application dependencies are compiled on the machine they are downloaded, many of them are written in GO source code. This project relies on some dependencies that may take a significant amount of time to compile on lower-spec machines! During this time it is not uncommon for that machine to become completely unresponsive!
### Manual
```shell
# Clone the repository
cd some/dir/to/clone/repo/to
git clone git@github.com:MartinMaybe/CS408-Project.git
cd CS408-Project

# Download go dependencies
go mod download

# Run the automated testing
go test .

# Run the application
go run .
```
After running the application, the server should be listening on port `8080`
### EC2 Deployment
```shell
# Clone the repository
cd some/dir/to/clone/repo/to
git clone git@github.com:MartinMaybe/CS408-Project.git
cd CS408-Project

# Run the EC2 configuration script
./scripts/ec2-config.sh

# Run the EC2 deployment script
./scripts/ec2-deploy.sh
```
>[!TIP]
>The EC2 scripts will automatically install all needed packages onto that EC2 instance & configure njinx to forward traffic accordingly.
## Technology Stack

- Backend technology stack
    - Web Server: [nginx](https://www.nginx.com/) as a reverse proxy server
    - Backend Runtime: [Go](https://go.dev/) go runtime
    - Backend Framework: [Go net/http](https://pkg.go.dev/net/http) go standard library
    - Database: [SQLite](https://www.sqlite.org/index.html) for lightweight data storage
- Frontend technology stack
    - Templates: [Go html/template]([https://ejs.co/](https://pkg.go.dev/html/template)) for server-side templating with go
    - UX/UI: [Bootstrap](https://getbootstrap.com/) for responsive design
- Testing Frameworks
    - Unit-Testing: [Go testing](https://pkg.go.dev/testing)
## API Endpoints
>[!TIP]
>The main webpages are entirely separate endpoints from the API. And are not qualified by the `/api` routing that all api endpoints have in common.

| **Method** | **Endpoint**                                | **Description**                                                                                                                        |
| ---------- | ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `POST`     | `/api/sessions`                             | Creates a new decision-tree session at the root node. Returns that `session_id`.                                                       |
| ``GET``    | `/api/sessions?session_id={id}`             | Grabs the current state of the session that corresponds to that particular `session_id`.                                               |
| `POST`     | `/api/session`                              | Provided with a corresponding `port_id` & `session_id`, will validate and advance that session by having it follow the specified port. |
| `GET`      | `/api/session/history?session_id={id}`      | Returns the recorded path for the given `session_id`, Includes each traversed node and the selected port between them.                 |
| `GET`      | `/api/node?node_id={id}`                    | Fetch a node corresponding to the given `node_id`.                                                                                     |
| `POST`     | `/api/node`                                 | Creates a new node. Provided the kind, prompt, and json body. The new `node_id` is returned.                                           |
| `GET`      | `/api/port?node_id={id}&port_key={yes\|no}` | Looks up a port ID that corresponds to a particular `node_id` and provided `key`                                                       |
| `POST`     | `/api/port`                                 | Attaches an existing *dangling* port to a node, provided a `port_id` & `to_node_id`.                                                   |

## Team Workflow
We use a single repo with both team members as collaborators. Workflow follows a by-checkpoint alternating contribution development system, where one team-member does 100% of the work for a single checkpoint, then it alternates to the next team-member who does 100% of the work for the following checkpoint.
## Stress Testing

>[!WARNING]
>Stress testing is an intensive process. Doing so on a machine that is ill-suited can be incredibly time consuming. During the time of stress testing, the service may experience diminished responsiveness!

Stress testing allows the developer to view the capabilities of the website, as if thousands of people were interacting with it. Stress testing uses a probabilistic model that is weighted by the question's contents and current time to randomly select answers & generate new questions when a particular path is exhausted.
### Usage
General usage is prompted by running:
```shell
go run <application directory> stress <flags>
```
*To view all available flags from terminal, you can use 
```shell
go run . stress -help
```
The default variation is ran with the following flags:
```shell
go run <app dir> -base-url "http://localhost:8080" -create-on-terminal=true -max-depth 25 -progress=true -seed <random default seed> -sessions 100 -timeout 10 -verbose=true
```

### Flags
- `base-url` : The url to the website from the point of where the stress test is being ran.
- `create-on-terminal` : Whether the stress test should attach and create new nodes on dangling branches.
- `delay`: A delay applied between API operations (in seconds).
- `max-depth` : The maximum number of decisions that any particular test session should generate.
- `progress` : Redraws a live progress bar at the bottom of the terminal output.
- `seed` : The random seed used for this stress run.
- `sessions` : The number of sessions to create over this stress run.
- `timeout` : HTTP client timeout.
- `verbose` : Print every API interaction to console as it is performed.

>[!TIP]
>Stress testing expects there to be an actively running session at the destination URL. Make sure that the application is currently running before attempting to use stress testing!
