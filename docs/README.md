# Full-Stack Web Application

This repository contains a full-stack web application built with Node.js,
Express, and SQLite. It includes scripts and documentation for setting up,
configuring, and deploying the application on an AWS EC2 instance. The application
uses Docker for containerization and simplified deployment.

- [Development Guide (Docker)](dev-node/README.md)
- [Deployment Guide (Docker)](deploy-docker/README.md)

>[!TIP]
>Docker is the recommended approach for development and deployment as it abstracts away many of the complexities of server configuration and application setup. It allows you to run the application in a consistent environment across different machines and simplifies the deployment process. You are not required to use Docker for this course, but it is highly recommended for a smoother experience. If you choose to deploy manually without Docker, you will need to follow the manual deployment instructions provided in the documentation.

## Technology Stack

- Backend technology stack
    - Web Server: [nginx](https://www.nginx.com/) as a reverse proxy server
    - Backend Runtime: [Node.js](https://nodejs.org/)
    - Backend Framework: [Express](https://expressjs.com/)
    - Database: [SQLite](https://www.sqlite.org/index.html) for lightweight data storage
- Frontend technology stack
    - Templates: [EJS](https://ejs.co/) for server-side rendering
    - UX/UI: [Bootstrap](https://getbootstrap.com/) for responsive design
- Testing Frameworks
    - End-to-End Testing: [Playwright](https://playwright.dev/)

## Manual Deployment

>[!WARNING]
>These instructions are for students who want to deploy the application manually without using Docker. This is an optional approach and is not required for the course. Manual deployment involves more steps and requires a deeper understanding of server configuration and application setup. It is recommended for students who want to gain hands-on experience with server administration and application deployment beyond the scope of Docker.

- [Deployment Guide (Manual)](deploy-node/README.md)
- [AWS EC2 Launch Guide](aws/README.md)

## Team Workflow

Single repo with collaborators, issues/stories are made on (TBD) scrumboard, issues are operated on with seperate branches and PR's used to implement changes on main.