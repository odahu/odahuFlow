#
#    Copyright 2018 EPAM Systems
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#
"""
odahuflow k8s definitions functions
"""

API_VERSION = 'v1'
CONFIGURATION_URL = '/api/{version}/configuration'
CONNECTION_URL = '/api/{version}/connection'
MODEL_TRAINING_URL = '/api/{version}/model/training'
TOOLCHAIN_INTEGRATION_URL = '/api/{version}/toolchain/integration'
MODEL_DEPLOYMENT_URL = '/api/{version}/model/deployment'
MODEL_DEPLOYMENT_DEFAULT_ROUTE_URL = '/api/{version}/model/deployment/{id}/default-route'
MODEL_ROUTE_URL = '/api/{version}/model/route'
MODEL_PACKING_URL = '/api/{version}/model/packaging'
PACKING_INTEGRATION_URL = '/api/{version}/packaging/integration'
INFERENCE_SERVICE_URL = '/api/{version}/batch/service'
INFERENCE_JOB_URL = '/api/{version}/batch/job'
FEEDBACK_URL = '/api/{version}/feedback'
USER_INFO_URL = '/api/{version}/user/info'
