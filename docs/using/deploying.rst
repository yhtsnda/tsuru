.. Copyright 2016 tsuru authors. All rights reserved.
   Use of this source code is governed by a BSD-style
   license that can be found in the LICENSE file.

Deploying
=========

Application requirements
------------------------

To your application supports horizontal scale it's recommended that the
application follow the `12 Factor <http://www.12factor.net/>`_ principles.

For example, if your application uses local memory to store session data it
should not works as expected with more than one `unit`.

Select a deployment process
---------------------------

tsuru supports three ways of deployment:

Git
+++

Git deployments are based on tsuru `platforms` and are useful if you want to
track the diference betwen the deployments.

:doc:`Learn how to deploy applications using Git </using/quickstart>`.

app-deploy
++++++++++

The `app-deploy` deployments are based on tsuru `platforms` and are useful for
automated deploymens.

Docker image
++++++++++++

Docker image deployments allows you to take a Docker image from a registry
ensuring that you are running the same image in development and in production.

:doc:`Learn how to deploy applications using Docker images </using/docker-image>`.
