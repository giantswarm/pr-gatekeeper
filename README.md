# General template repository

This is a general template repository containing some basic files every GitHub repo owned by Giant Swarm should have.

Note also these more specific repositories:

- [template-app](https://github.com/giantswarm/template-app)
- [gitops-template](https://github.com/giantswarm/gitops-template)
- [python-app-template](https://github.com/giantswarm/python-app-template)

To use this template, just hit the **Use this template** button above or click [this link](https://github.com/giantswarm/template/generate).

## Adapting your repository

After you created your new repo with name `REPOSTORY_NAME`, please follow these steps:

1. Configure the correct repository name in all files:

   ```nohighlight
   devctl replace -i REPOSTORY_NAME \
     "$(basename $(git rev-parse --show-toplevel))" --ignore '.git/**' '**'
     
   devctl replace -i "template" \
     "$(basename $(git rev-parse --show-toplevel))" --ignore '.git/**' '**'
   ```
   
   Find `devctl` [here](https://github.com/giantswarm/devctl).

2. Adjust repo settings in the [repo settings page](https://github.com/giantswarm/REPOSTORY_NAME/settings). Make sure that the
   - **Allow merge commits** box is not checked
   - **Automatically delete head branches** box is checked

3. Adjust access permissions on the [access settings page](https://github.com/giantswarm/REPOSTORY_NAME/settings/access) as follows:
   - Add `giantswarm/bots` with `Write` access.
   - Add `giantswarm/employees` with `Admin` access.

4. Add this repository to your team's [repositories list](https://github.com/giantswarm/github/tree/master/repositories) in the giantswarm/github repository, to keep up-to-date with general changes.

5. Replace these instructions by meaningful `README.md` content.

6. Adjust the repository description, tags, and under "Include in the home page" deselect the Packages and Environments options.

### Optional

- If a container image will get built based on your repository, [set up a Quay.io repository](https://intranet.giantswarm.io/docs/dev-and-releng/container-registry/) for it.

- Add the project to the CircleCI via [this link](https://circleci.com/setup-project/gh/giantswarm/REPOSTORY_NAME)

- Add badges to the top of `README.md` where applicable

  - [CircleCI](https://app.circleci.com/settings/project/github/giantswarm/REPOSTORY_NAME/status-badges). Note: if this is a private repository, an [API token](https://app.circleci.com/settings/project/github/giantswarm/REPOSTORY_NAME/api) with scope `status` will be needed. The resulting badge URL will look something like `https://circleci.com/gh/giantswarm/REPOSTORY_NAME.svg?style=svg&circle-token=TOKEN_FOR_PRIVATE_REPO`.
  
  - [Quay.io](https://quay.io/repository/giantswarm/REPOSTORY_NAME?tab=settings)
   
   

