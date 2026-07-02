from vetix.utils.utils import get_version


def print_banner():
    print(r"""
   _    __     __  _     
  | |  / /__  / /_(_)  __
  | | / / _ `/ __/ / `/_/
  | |/ /  __/ /_/ />  <  
  |___/\___/\__/_/_/|_|   """ + get_version() + "\n")
