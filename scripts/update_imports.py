#!/usr/bin/env python3

import os
import re

def update_imports():
    old_module = "github.com/growthfolio/go-priceguard-api"
    new_module = "github.com/growthfolio/go-priceguard-api"
    
    # Find all .go files
    for root, dirs, files in os.walk("."):
        # Skip vendor directory
        if "vendor" in dirs:
            dirs.remove("vendor")
            
        for file in files:
            if file.endswith(".go"):
                file_path = os.path.join(root, file)
                
                try:
                    with open(file_path, 'r', encoding='utf-8') as f:
                        content = f.read()
                    
                    if old_module in content:
                        print(f"Updating: {file_path}")
                        new_content = content.replace(old_module, new_module)
                        
                        with open(file_path, 'w', encoding='utf-8') as f:
                            f.write(new_content)
                            
                except Exception as e:
                    print(f"Error processing {file_path}: {e}")

if __name__ == "__main__":
    update_imports()
    print("Import update complete!")
