import sys
import os
import subprocess

# Path to the manager.py script
MANAGER_SCRIPT = os.path.join(os.path.dirname(__file__), "manager.py")

def main():
    if len(sys.argv) < 2:
        print("Usage: python backend.py <manager_command> [args...]", file=sys.stderr)
        sys.exit(1)

    manager_command = sys.argv[1]
    manager_args = sys.argv[2:]

    full_command = ["python", MANAGER_SCRIPT, manager_command] + manager_args

    try:
        # Execute the manager.py script
        # We don't capture output here, as manager.py writes to status.json
        subprocess.run(full_command, check=True, text=True, encoding='utf-8', 
                       stdout=sys.stderr, stderr=sys.stderr) 
        # Redirect stdout/stderr of manager.py to backend.py's stderr for debugging.
        # Manager.py now writes status to status.json
    except subprocess.CalledProcessError as e:
        print(f"Error executing manager.py: {e}", file=sys.stderr)
        sys.exit(e.returncode)
    except Exception as e:
        print(f"Error unexpected in backend.py: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()